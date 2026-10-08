// social_revocation_repo.go — Queries against ALUMNI_SOCIAL_REVOCATION_OUTBOX
// used exclusively by the background drain worker
// (internal/job/social_revocation_worker.go). The synchronous disconnect path
// enqueues rows here but never reads them back; only the worker claims and
// finalizes them. ACCOUNT_DELETE rows may remain from the earlier deletion flow.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrSocialRevocationClaimExpired = errors.New("social revocation claim is no longer current")

// Lock the connection before the outbox, matching linking's lock order. A
// slow worker cannot read a replacement credential or revoke while relinking
// commits. Provider calls use a bounded context supplied by the worker.
func lockClaimedDisconnect(tx *sqlx.Tx, entry model.SocialRevocationOutboxEntry, claimToken string) (string, string, error) {
	var connectionStatus string
	err := tx.Get(&connectionStatus, `SELECT NMS_STATUS FROM WEO_MEMBER_SOCIAL
        WHERE USR_SEQ=? AND NMS_GATE=? FOR UPDATE`, entry.USRSeq, entry.Provider)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", "", err
	}
	var outbox struct {
		Status string `db:"STATUS"`
		Claim  string `db:"CLAIM_TOKEN"`
	}
	err = tx.Get(&outbox, `SELECT STATUS, COALESCE(CLAIM_TOKEN, '') AS CLAIM_TOKEN
        FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX
        WHERE OUTBOX_ID=? AND USR_SEQ=? AND PROVIDER=? AND ACTION='DISCONNECT' FOR UPDATE`, entry.OutboxID, entry.USRSeq, entry.Provider)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrSocialRevocationClaimExpired
	}
	if err != nil {
		return "", "", err
	}
	if claimToken == "" || outbox.Claim != claimToken || (outbox.Status != "PENDING" && outbox.Status != "REVOKED") ||
		(connectionStatus != "" && connectionStatus != "DISCONNECTING" && connectionStatus != "FINALIZE_PENDING") {
		return "", "", ErrSocialRevocationClaimExpired
	}
	return connectionStatus, outbox.Status, nil
}

// The provider checkpoint commits before local finalization. If the provider
// succeeded but the checkpoint write failed, the caller retries only finalization.
func (r *AuthRepository) RevokeClaimedSocialDisconnect(ctx context.Context, entry model.SocialRevocationOutboxEntry, claimToken string, revoke func(string) error) (bool, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	connection, status, err := lockClaimedDisconnect(tx, entry, claimToken)
	if err != nil {
		return false, err
	}
	if status == "REVOKED" {
		return true, tx.Commit()
	}
	if connection != "DISCONNECTING" {
		return false, ErrSocialRevocationClaimExpired
	}
	var encrypted string
	if err := tx.Get(&encrypted, `SELECT ENCRYPTED_CREDENTIAL FROM ALUMNI_SOCIAL_CREDENTIAL
        WHERE USR_SEQ=? AND PROVIDER=?`, entry.USRSeq, entry.Provider); err != nil {
		return false, err
	}
	if err := revoke(encrypted); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX SET STATUS='REVOKED', UPDATED_AT=NOW()
        WHERE OUTBOX_ID=? AND CLAIM_TOKEN=?`, entry.OutboxID, claimToken); err != nil {
		return true, err
	}
	return true, tx.Commit()
}

func (r *AuthRepository) FinalizeClaimedSocialDisconnect(entry model.SocialRevocationOutboxEntry, claimToken string) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, status, err := lockClaimedDisconnect(tx, entry, claimToken)
	if err != nil {
		return err
	}
	if status != "REVOKED" {
		return ErrSocialRevocationClaimExpired
	}
	if err := r.deleteSocialConnectionTx(tx, entry.USRSeq, entry.Provider, true, true); err != nil {
		return err
	}
	return tx.Commit()
}

// ClaimDueSocialRevocations atomically checks out up to limit due outbox rows
// (STATUS IN PENDING/REVOKED, NEXT_ATTEMPT_AT <= NOW()) by writing claimToken
// into CLAIM_TOKEN, then returns exactly the rows it claimed.
//
// This exists so that two concurrent worker processes (e.g. an overlapping
// deploy restart) cannot both fetch and process the same row - each caller's
// claiming UPDATE only matches rows not already claimed by a still-live claim,
// so at most one caller "wins" a given row. A row whose claim is older than
// staleAfter is treated as abandoned (the worker that claimed it crashed
// before finishing) and can be re-claimed, so a crash never strands a row
// forever.
//
// Includes both PENDING (upstream provider not yet revoked) and REVOKED
// (upstream already revoked, only local finalization remains) rows - see
// MarkSocialRevocationFailed for why these two states must be retried
// differently.
func (r *AuthRepository) ClaimDueSocialRevocations(claimToken string, staleAfter time.Duration, limit int) ([]model.SocialRevocationOutboxEntry, error) {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET CLAIM_TOKEN = ?, UPDATED_AT = NOW()
		WHERE STATUS IN ('PENDING', 'REVOKED')
		  AND NEXT_ATTEMPT_AT <= NOW()
		  AND (CLAIM_TOKEN IS NULL OR UPDATED_AT <= NOW() - INTERVAL ? SECOND)
		ORDER BY NEXT_ATTEMPT_AT
		LIMIT ?
	`, claimToken, int(staleAfter.Seconds()), limit)
	if err != nil {
		return nil, err
	}

	// LAST_ERROR is nullable in the schema; rows created without an error
	// message must still be claimable. Rows are read one by one so a single
	// unreadable row is reported instead of stalling the whole batch.
	rows, err := r.DB.Queryx(`
		SELECT OUTBOX_ID, USR_SEQ, PROVIDER, ACTION, STATUS, ATTEMPT_COUNT,
		       NEXT_ATTEMPT_AT, COALESCE(LAST_ERROR, '') AS LAST_ERROR, CREATED_AT, UPDATED_AT
		FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX
		WHERE CLAIM_TOKEN = ?
		ORDER BY NEXT_ATTEMPT_AT
	`, claimToken)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []model.SocialRevocationOutboxEntry
	skipped := 0
	var firstErr error
	for rows.Next() {
		var entry model.SocialRevocationOutboxEntry
		if err := rows.StructScan(&entry); err != nil {
			skipped++
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return entries, err
	}
	if skipped > 0 {
		return entries, fmt.Errorf("skipped %d unreadable social revocation rows: %w", skipped, firstErr)
	}
	return entries, nil
}

// MarkSocialRevocationRevoked durably records that the upstream provider has
// already revoked/unlinked the credential, BEFORE the worker attempts local
// finalization (MarkSocialDisconnectRevoked+DeleteSocialConnection for
// DISCONNECT, or CompleteAccountDeletionRevocation for ACCOUNT_DELETE). This
// is the sole checkpoint that prevents a retry from ever calling the
// provider's revoke/unlink API a second time: no matter what fails afterward
// (finalization, or a crash), the next attempt reads STATUS=REVOKED from the
// database and only retries local finalization.
//
// Deliberately does NOT clear CLAIM_TOKEN: this call happens mid-attempt,
// immediately followed by finalization within the same processEntry
// invocation, so releasing the claim here would reopen the exact
// concurrent-processing window ClaimDueSocialRevocations exists to close. The
// claim is only released by a terminal write (MarkSocialRevocationSucceeded
// or MarkSocialRevocationFailed) or, if the worker crashes before either
// runs, once it goes stale.
func (r *AuthRepository) MarkSocialRevocationRevoked(outboxID int64) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET STATUS = 'REVOKED', UPDATED_AT = NOW()
		WHERE OUTBOX_ID = ?
	`, outboxID)
	return err
}

// FinalizeSocialDisconnect completes a DISCONNECT entry after the worker has
// confirmed the upstream provider revoke succeeded (directly, or via a prior
// MarkSocialRevocationRevoked checkpoint). It is the worker's idempotent
// counterpart to social_account_lifecycle.go's synchronous disconnect
// finalization (MarkSocialDisconnectRevoked + DeleteSocialConnection): unlike
// that path, this one must tolerate being retried after a crash at any point,
// since the worker may reattempt the exact same finalize step more than once.
//
//   - If WEO_MEMBER_SOCIAL is still DISCONNECTING, advance it to
//     FINALIZE_PENDING (the normal case).
//   - If it's already FINALIZE_PENDING, or the row is already gone (a prior
//     attempt's DeleteSocialConnection already ran), treat as already
//     advanced and proceed - DeleteSocialConnection's own DELETE/UPDATE
//     statements are plain WHERE-matched and safely no-op on already-cleaned
//     state.
//   - Any other NMS_STATUS (e.g. ACTIVE) is a genuine anomaly and returns an
//     error rather than silently overwriting it.
func (r *AuthRepository) FinalizeSocialDisconnect(usrSeq int, provider string) error {
	result, err := r.DB.Exec(`
		UPDATE WEO_MEMBER_SOCIAL
		SET NMS_STATUS = 'FINALIZE_PENDING'
		WHERE USR_SEQ = ? AND NMS_GATE = ? AND NMS_STATUS = 'DISCONNECTING'
	`, usrSeq, provider)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		var currentStatus string
		err := r.DB.Get(&currentStatus, `
			SELECT NMS_STATUS FROM WEO_MEMBER_SOCIAL
			WHERE USR_SEQ = ? AND NMS_GATE = ?
		`, usrSeq, provider)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// Already deleted by a prior finalize attempt - nothing to advance.
		case err != nil:
			return err
		case currentStatus != "FINALIZE_PENDING":
			return errors.New("social disconnect finalization found an unexpected link state: " + currentStatus)
		}
	}
	return r.deleteSocialConnection(usrSeq, provider, true, true)
}

// MarkSocialRevocationSucceeded releases the claim on a row that the
// worker's own finalization step (MarkSocialDisconnectRevoked+
// DeleteSocialConnection, or CompleteAccountDeletionRevocation) has already
// set to STATUS='DELIVERED'. Those methods own the DELIVERED transition
// themselves (matching the synchronous disconnect path's behavior); this only
// clears CLAIM_TOKEN so the row is no longer considered claimed.
func (r *AuthRepository) MarkSocialRevocationSucceeded(outboxID int64, claimToken ...string) error {
	claimCondition, args := socialRevocationClaimCondition(claimToken)
	args = append([]any{outboxID}, args...)
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET CLAIM_TOKEN = NULL, UPDATED_AT = NOW()
		WHERE OUTBOX_ID = ?
	`+claimCondition, args...)
	return err
}

// MarkSocialRevocationFailed records a failed revocation attempt, setting
// ATTEMPT_COUNT to newAttemptCount (the caller's post-increment count). The
// entry is reset to retryStatus (to retry at nextAttempt) unless
// newAttemptCount has reached maxAttempts, in which case it is marked FAILED
// (terminal, but left in the table for operator visibility).
//
// retryStatus must be "PENDING" when the failure happened before the
// upstream provider call succeeded (so the next attempt retries the full
// revoke-then-finalize flow), or "REVOKED" when the upstream provider was
// already successfully revoked and only local finalization failed - retrying
// as PENDING in that case would call Kakao/Apple unlink again on an
// already-revoked credential, which providers reject, permanently stranding
// the row.
//
// Clears only the matching claim when supplied, and never resets a DELIVERED
// row. Whether the retry stays PENDING/REVOKED or becomes terminal FAILED,
// re-selection is gated by NEXT_ATTEMPT_AT/STATUS.
func (r *AuthRepository) MarkSocialRevocationFailed(outboxID int64, errMsg string, newAttemptCount int, maxAttempts int, nextAttempt time.Time, retryStatus string, claimToken ...string) error {
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	status := retryStatus
	if newAttemptCount >= maxAttempts {
		status = "FAILED"
		if retryStatus == "REVOKED" {
			status = "FINALIZE_FAILED"
		}
	}
	claimCondition, args := socialRevocationClaimCondition(claimToken)
	args = append([]any{status, newAttemptCount, errMsg, nextAttempt, outboxID}, args...)
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET STATUS = ?, CLAIM_TOKEN = NULL, ATTEMPT_COUNT = ?, LAST_ERROR = ?,
		    NEXT_ATTEMPT_AT = ?, UPDATED_AT = NOW()
		WHERE OUTBOX_ID = ? AND STATUS <> 'DELIVERED'
	`+claimCondition, args...)
	return err
}

func socialRevocationClaimCondition(claimToken []string) (string, []any) {
	if len(claimToken) == 0 {
		return "", nil
	}
	return " AND CLAIM_TOKEN = ?", []any{claimToken[0]}
}
