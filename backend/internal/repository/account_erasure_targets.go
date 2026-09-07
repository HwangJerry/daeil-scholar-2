// account_erasure_targets.go — Durable partial progress and monotonic verification.
package repository

import (
	"database/sql"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
	"strings"
)

func seedErasureTargets(tx *sqlx.Tx, id int64) error {
	for _, name := range model.ErasureTargetNames {
		if _, err := tx.Exec(`INSERT IGNORE INTO ALUMNI_ERASURE_TARGET (REQUEST_ID,TARGET,UPDATED_AT) VALUES (?,?,UTC_TIMESTAMP())`, id, name); err != nil {
			return err
		}
	}
	return nil
}
func (r *AccountDeletionRequestRepository) ErasureTargets(id int64) ([]model.ErasureTarget, error) {
	targets := []model.ErasureTarget{}
	err := r.DB.Select(&targets, `SELECT TARGET,STATUS,EVIDENCE_REFERENCE,LAST_CODE,ATTEMPTS,LAST_ATTEMPT_AT,UPDATED_AT FROM ALUMNI_ERASURE_TARGET WHERE REQUEST_ID=? ORDER BY TARGET`, id)
	for i := range targets {
		targets[i].UpdatedAt = deletionTimeUTC(targets[i].UpdatedAt)
		if targets[i].LastAttemptAt != nil {
			v := deletionTimeUTC(*targets[i].LastAttemptAt)
			targets[i].LastAttemptAt = &v
		}
	}
	return targets, err
}
func (r *AccountDeletionRequestRepository) BeginErasureTargets(id int64) error {
	result, err := r.DB.Exec(`UPDATE ALUMNI_ERASURE_TARGET t JOIN ALUMNI_ACCOUNT_ERASURE e ON e.REQUEST_ID=t.REQUEST_ID
 SET t.STATUS='running',t.ATTEMPTS=t.ATTEMPTS+1,t.LAST_ATTEMPT_AT=UTC_TIMESTAMP(),t.UPDATED_AT=UTC_TIMESTAMP()
 WHERE t.REQUEST_ID=? AND e.MODE='automatic' AND t.STATUS NOT IN ('complete','not_applicable')`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *AccountDeletionRequestRepository) RecordErasureTargets(id int64, targets []model.ErasureTarget) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var mode string
	if err = tx.Get(&mode, `SELECT MODE FROM ALUMNI_ACCOUNT_ERASURE WHERE REQUEST_ID=? AND STAGE<>'completed' FOR UPDATE`, id); err != nil {
		return err
	}
	if mode != "automatic" {
		return sql.ErrNoRows
	}
	seen := map[string]bool{}
	for _, target := range targets {
		validName := false
		for _, name := range model.ErasureTargetNames {
			if target.Name == name {
				validName = true
			}
		}
		if !validName || seen[target.Name] || len(target.Evidence) > 200 {
			return ErrDeletionIncomplete
		}
		seen[target.Name] = true
		switch target.Status {
		case "pending", "failed", "manual":
		case "complete", "not_applicable":
			if strings.TrimSpace(target.Evidence) == "" {
				return ErrDeletionIncomplete
			}
		default:
			return ErrDeletionIncomplete
		}
		// Code values are owned by our worker, never raw remote errors.
		code := ""
		switch target.Status {
		case "pending":
			code = "EXTERNAL_ERASURE_PENDING"
		case "failed":
			code = "EXTERNAL_ERASURE_RETRY_REQUIRED"
		case "manual":
			code = "EXTERNAL_ERASURE_REVIEW_REQUIRED"
		}
		if _, err = tx.Exec(`UPDATE ALUMNI_ERASURE_TARGET SET STATUS=?,EVIDENCE_REFERENCE=?,LAST_CODE=?,UPDATED_AT=UTC_TIMESTAMP()
 WHERE REQUEST_ID=? AND TARGET=? AND STATUS NOT IN ('complete','not_applicable')`, target.Status, target.Evidence, code, id, target.Name); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func verifiedErasureTargets(tx *sqlx.Tx, id int64) (bool, error) {
	var count int
	err := tx.Get(&count, `SELECT COUNT(*) FROM ALUMNI_ERASURE_TARGET WHERE REQUEST_ID=?
 AND TARGET IN ('backups','historical_files','external_data','other_identifiers')
 AND STATUS IN ('complete','not_applicable') AND EVIDENCE_REFERENCE<>''`, id)
	return count == len(model.ErasureTargetNames), err
}
