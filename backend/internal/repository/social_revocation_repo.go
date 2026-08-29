// social_revocation_repo.go — Queries against ALUMNI_SOCIAL_REVOCATION_OUTBOX
// used exclusively by the background drain worker
// (internal/job/social_revocation_worker.go). The synchronous disconnect path
// enqueues rows here but never reads them back; only the worker claims and
// finalizes them. ACCOUNT_DELETE rows may remain from the earlier deletion flow.
package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

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

	var entries []model.SocialRevocationOutboxEntry
	err = r.DB.Select(&entries, `
		SELECT OUTBOX_ID, USR_SEQ, PROVIDER, ACTION, STATUS, ATTEMPT_COUNT,
		       NEXT_ATTEMPT_AT, LAST_ERROR, CREATED_AT, UPDATED_AT
		FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX
		WHERE CLAIM_TOKEN = ?
		ORDER BY NEXT_ATTEMPT_AT
	`, claimToken)
	return entries, err
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
	return r.DeleteSocialConnection(usrSeq, provider)
}

// MarkSocialRevocationSucceeded releases the claim on a row that the
// worker's own finalization step (MarkSocialDisconnectRevoked+
// DeleteSocialConnection, or CompleteAccountDeletionRevocation) has already
// set to STATUS='DELIVERED'. Those methods own the DELIVERED transition
// themselves (matching the synchronous disconnect path's behavior); this only
// clears CLAIM_TOKEN so the row is no longer considered claimed.
func (r *AuthRepository) MarkSocialRevocationSucceeded(outboxID int64) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET CLAIM_TOKEN = NULL, UPDATED_AT = NOW()
		WHERE OUTBOX_ID = ?
	`, outboxID)
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
// Always clears CLAIM_TOKEN: whether the retry stays PENDING/REVOKED (due at
// nextAttempt) or becomes terminal FAILED, re-selection is already gated by
// NEXT_ATTEMPT_AT/STATUS, so nothing depends on the claim staying held.
func (r *AuthRepository) MarkSocialRevocationFailed(outboxID int64, errMsg string, newAttemptCount int, maxAttempts int, nextAttempt time.Time, retryStatus string) error {
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	status := retryStatus
	if newAttemptCount >= maxAttempts {
		status = "FAILED"
	}
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET STATUS = ?, CLAIM_TOKEN = NULL, ATTEMPT_COUNT = ?, LAST_ERROR = ?,
		    NEXT_ATTEMPT_AT = ?, UPDATED_AT = NOW()
		WHERE OUTBOX_ID = ?
	`, status, newAttemptCount, errMsg, nextAttempt, outboxID)
	return err
}
