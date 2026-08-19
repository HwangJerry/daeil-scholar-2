// social_revocation_repo.go — Queries against ALUMNI_SOCIAL_REVOCATION_OUTBOX,
// the outbox drained by the social revocation background worker
// (internal/job/social_revocation_worker.go) to actually call out to Kakao/Apple
// and revoke provider tokens after a disconnect or account deletion.
package repository

import (
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

// ClaimDueSocialRevocations atomically checks out up to limit due outbox rows
// (STATUS IN PENDING/REVOKED, NEXT_ATTEMPT_AT <= NOW()) by writing claimToken
// into CLAIM_TOKEN, then returns exactly the rows it claimed.
//
// This exists so that two concurrent worker processes (e.g. an overlapping
// deploy restart, or a misconfigured second instance) cannot both fetch and
// process the same row - each caller's claiming UPDATE only matches rows not
// already claimed by a still-live claim, so at most one caller "wins" a given
// row. A row whose claim is older than staleAfter is treated as abandoned
// (the worker that claimed it crashed before finishing) and can be
// re-claimed, so a crash never strands a row forever.
//
// Includes both PENDING (upstream provider not yet revoked) and REVOKED
// (upstream already revoked, only the local
// WEO_MEMBER_SOCIAL/ALUMNI_SOCIAL_CREDENTIAL cleanup remains) rows - see
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

// MarkSocialRevocationSucceeded marks an outbox entry as done after the
// upstream provider revocation call has succeeded. Clears CLAIM_TOKEN since
// this is a terminal write - the row is finished, nothing will claim it again.
func (r *AuthRepository) MarkSocialRevocationSucceeded(outboxID int64) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET STATUS = 'DONE', CLAIM_TOKEN = NULL, UPDATED_AT = NOW()
		WHERE OUTBOX_ID = ?
	`, outboxID)
	return err
}

// MarkSocialRevocationRevoked durably records that the upstream provider has
// already revoked/unlinked the credential, BEFORE the worker attempts local
// cleanup (DeleteSocialConnection) or marks the entry DONE. This is the sole
// checkpoint that prevents a retry from ever calling the provider's
// revoke/unlink API a second time: no matter what fails afterward (the local
// delete, the DONE write, or a crash), the next attempt reads STATUS=REVOKED
// from the database and only retries the local cleanup.
//
// Deliberately does NOT clear CLAIM_TOKEN: this call happens mid-attempt,
// immediately followed by DeleteSocialConnection and MarkSocialRevocationSucceeded
// within the same processEntry invocation, so releasing the claim here would
// reopen the exact concurrent-processing window ClaimDueSocialRevocations
// exists to close. The claim is only released by a terminal write
// (MarkSocialRevocationSucceeded or MarkSocialRevocationFailed) or, if the
// worker crashes before either runs, once it goes stale.
func (r *AuthRepository) MarkSocialRevocationRevoked(outboxID int64) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET STATUS = 'REVOKED', UPDATED_AT = NOW()
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
// revoke-then-cleanup flow), or "REVOKED" when the upstream provider was
// already successfully revoked and only the local
// WEO_MEMBER_SOCIAL/ALUMNI_SOCIAL_CREDENTIAL cleanup failed - retrying as
// PENDING in that case would call Kakao/Apple unlink again on an
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
