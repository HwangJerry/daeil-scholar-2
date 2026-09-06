// privacy_retention_repo.go — Bounded cleanup of resolved reports and completed receipts.
package repository

import "context"

// Unresolved reports and incomplete deletion requests are never expired here.
// A lawful preservation order must be handled in the separate restricted legal
// archive before a report is marked resolved (see the operator runbook).
func (r *AccountDeletionRequestRepository) PurgeExpiredPrivacyRecords(ctx context.Context, limit int) error {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if _, err := r.DB.ExecContext(ctx, `DELETE FROM ALUMNI_MESSAGE_REPORT
        WHERE STATUS IN ('removed', 'dismissed') AND RESOLVED_AT < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 90 DAY)
        ORDER BY RESOLVED_AT LIMIT ?`, limit); err != nil {
		return err
	}
	_, err := r.DB.ExecContext(ctx, `DELETE FROM ALUMNI_ACCOUNT_DELETION_REQUEST
        WHERE STATUS = 'completed' AND COMPLETED_AT < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 30 DAY)
        ORDER BY COMPLETED_AT LIMIT ?`, limit)
	return err
}
