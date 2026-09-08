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
	var contextInstalled int
	if err := r.DB.GetContext(ctx, &contextInstalled, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_ERASURE_CONTEXT'`); err != nil {
		return err
	}
	if contextInstalled > 0 {
		if _, err := r.DB.ExecContext(ctx, `DELETE FROM ALUMNI_ERASURE_CONTEXT WHERE EXPIRES_AT<=UTC_TIMESTAMP() ORDER BY EXPIRES_AT LIMIT ?`, limit); err != nil {
			return err
		}
	}
	// Optional during the rolling migration; the new worker requires migration 057.
	var installed int
	if err := r.DB.GetContext(ctx, &installed, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_DONATION_LEGAL_ARCHIVE'`); err != nil {
		return err
	}
	if installed > 0 {
		if _, err := r.DB.ExecContext(ctx, `DELETE FROM ALUMNI_DONATION_LEGAL_ARCHIVE WHERE RETAIN_UNTIL < DATE(DATE_ADD(UTC_TIMESTAMP(), INTERVAL 9 HOUR)) ORDER BY RETAIN_UNTIL LIMIT ?`, limit); err != nil {
			return err
		}
		if _, err := r.DB.ExecContext(ctx, `DELETE FROM ALUMNI_ACCOUNT_ERASURE WHERE EXISTS (SELECT 1 FROM ALUMNI_ACCOUNT_DELETION_REQUEST d WHERE d.REQUEST_ID=ALUMNI_ACCOUNT_ERASURE.REQUEST_ID AND d.STATUS='completed' AND d.COMPLETED_AT < DATE_SUB(UTC_TIMESTAMP(),INTERVAL 30 DAY)) ORDER BY REQUEST_ID LIMIT ?`, limit); err != nil {
			return err
		}
	}
	_, err := r.DB.ExecContext(ctx, `DELETE FROM ALUMNI_ACCOUNT_DELETION_REQUEST
        WHERE STATUS = 'completed' AND COMPLETED_AT < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 30 DAY)
        ORDER BY COMPLETED_AT LIMIT ?`, limit)
	return err
}
