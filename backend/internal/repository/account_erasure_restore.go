// account_erasure_restore.go — Guard completed erasures against backup restores and re-erase restored members.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// RestoreGuardDays covers the 28-day backup retention plus one weekly interval.
const RestoreGuardDays = 35

func recordRestoreGuard(tx *sqlx.Tx, id int64, user int) error {
	_, err := tx.Exec(`INSERT INTO ALUMNI_ERASURE_RESTORE_GUARD (REQUEST_ID,USR_SEQ,COMPLETED_AT,EXPIRES_AT)
        VALUES (?,?,UTC_TIMESTAMP(),DATE_ADD(UTC_TIMESTAMP(),INTERVAL ? DAY))
        ON DUPLICATE KEY UPDATE USR_SEQ=VALUES(USR_SEQ),COMPLETED_AT=VALUES(COMPLETED_AT),EXPIRES_AT=VALUES(EXPIRES_AT)`, id, user, RestoreGuardDays)
	return err
}

// PurgeExpiredRestoreGuards forgets erased member numbers once no retained
// backup can contain them any more.
func (r *AccountDeletionRequestRepository) PurgeExpiredRestoreGuards(ctx context.Context) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM ALUMNI_ERASURE_RESTORE_GUARD WHERE EXPIRES_AT < UTC_TIMESTAMP()`)
	return err
}

// RestoreGuards lists unexpired guards and whether each member row exists again.
func (r *AccountDeletionRequestRepository) RestoreGuards() ([]model.ErasureRestoreGuard, error) {
	guards := []model.ErasureRestoreGuard{}
	err := r.DB.Select(&guards, `SELECT g.REQUEST_ID, g.USR_SEQ, EXISTS(SELECT 1 FROM WEO_MEMBER m WHERE m.USR_SEQ=g.USR_SEQ) AS PRESENT
        FROM ALUMNI_ERASURE_RESTORE_GUARD g WHERE g.EXPIRES_AT >= UTC_TIMESTAMP() ORDER BY g.REQUEST_ID`)
	return guards, err
}

// ReapplyErasureAfterRestore erases a guarded member that a restored backup
// brought back, with the same rules as the worker, and returns the local files
// to unlink. Provider unlinks already happened, so restored revocation work is
// dropped. Donation rows need reviewed retention decisions and stop the run.
func (r *AccountDeletionRequestRepository) ReapplyErasureAfterRestore(user int) ([]string, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var guarded int
	if err = tx.Get(&guarded, `SELECT COUNT(*) FROM ALUMNI_ERASURE_RESTORE_GUARD WHERE USR_SEQ=? AND EXPIRES_AT >= UTC_TIMESTAMP()`, user); err != nil {
		return nil, err
	}
	if guarded == 0 {
		return nil, sql.ErrNoRows
	}
	s, err := readErasureSchema(tx)
	if err != nil {
		return nil, err
	}
	if s["WEO_ORDER"] != nil {
		var orders int
		if err = tx.Get(&orders, `SELECT COUNT(*) FROM WEO_ORDER WHERE USR_SEQ=? OR O_ACCOUNT_USR_SEQ=?`, user, user); err != nil {
			return nil, err
		}
		if orders > 0 {
			return nil, &model.ErasureBlocked{Code: "RESTORE_DONATION_REVIEW_REQUIRED"}
		}
	}
	email := ""
	if s.has("WEO_MEMBER", "USR_EMAIL") {
		if err = tx.Get(&email, `SELECT COALESCE(USR_EMAIL,'') FROM WEO_MEMBER WHERE USR_SEQ=?`, user); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	plan, err := planErasureFiles(tx, s, model.ErasureWork{UserSeq: user}, r.SiteOrigin)
	if err != nil {
		return nil, err
	}
	if err = erasePostChildren(tx, s, user); err != nil {
		return nil, err
	}
	if err = eraseAccountReferences(tx, s, user, email); err != nil {
		return nil, err
	}
	for _, id := range plan.fileIDs {
		if err = s.erase(tx, "WEO_FILES", "F_SEQ=?", id); err != nil {
			return nil, err
		}
	}
	if err = s.erase(tx, "ALUMNI_SOCIAL_REVOCATION_OUTBOX", "USR_SEQ=?", user); err != nil {
		return nil, err
	}
	if err = s.erase(tx, "WEO_MEMBER", "USR_SEQ=?", user); err != nil {
		return nil, err
	}
	remaining, err := deletionFootprint(tx, user)
	if err != nil {
		return nil, err
	}
	if len(remaining) > 0 {
		return nil, &model.ErasureBlocked{Code: "UNHANDLED_ACCOUNT_REFERENCE"}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return plan.locals, nil
}
