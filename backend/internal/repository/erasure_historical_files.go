// erasure_historical_files.go — Add reviewed historical files to the existing durable queue.
package repository

import (
	"crypto/sha256"
	"fmt"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// The CLI also holds ErasureLock, preventing the worker from draining a batch
// while this plan is being inspected. The transaction protects administrator
// takeover and final completion. Dry runs roll back without writing any rows.
func (r *AccountDeletionRequestRepository) QueueHistoricalErasureFiles(plan model.ErasureHistoricalFilePlan, apply bool) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var user int
	if err = tx.Get(&user, `SELECT USR_SEQ FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND STATUS='processing' FOR UPDATE`, plan.RequestID); err != nil {
		return err
	}
	if user != plan.UserSeq {
		return &model.ErasureBlocked{Code: "HISTORICAL_FILE_ACCOUNT_MISMATCH"}
	}
	var id int64
	if err = tx.Get(&id, `SELECT REQUEST_ID FROM ALUMNI_ACCOUNT_ERASURE WHERE REQUEST_ID=? AND MODE='automatic' AND STAGE='database_erased' AND EXTERNAL_EVIDENCE='' FOR UPDATE`, plan.RequestID); err != nil {
		return err
	}
	var status string
	if err = tx.Get(&status, `SELECT STATUS FROM ALUMNI_ERASURE_TARGET WHERE REQUEST_ID=? AND TARGET='historical_files' FOR UPDATE`, plan.RequestID); err != nil {
		return err
	}
	if status == "complete" || status == "not_applicable" {
		return ErrDeletionIncomplete
	}
	schema, err := readErasureSchema(tx)
	if err != nil {
		return err
	}
	for _, raw := range plan.Files {
		if err = rejectReferencedHistoricalFile(tx, schema, raw); err != nil {
			return err
		}
	}
	if !apply {
		return nil
	}
	for _, raw := range plan.Files {
		hash := sha256.Sum256([]byte(raw))
		if _, err = tx.Exec(`INSERT IGNORE INTO ALUMNI_ERASURE_FILE (REQUEST_ID,URL_PATH,URL_HASH) VALUES (?,?,?)`, plan.RequestID, raw, fmt.Sprintf("%x", hash)); err != nil {
			return err
		}
		if schema["WEO_FILES"] != nil {
			if err = schema.erase(tx, "WEO_FILES", "F_JOIN_SEQ=0 AND CONCAT(FILE_PATH,'/',FILE_NAME)=?", raw); err != nil {
				return err
			}
		}
	}
	// Evidence means reviewed inventory, NOT that files or all historical copies
	// are gone. Only the worker deletes files; scope completion still needs proof.
	if _, err = tx.Exec(`UPDATE ALUMNI_ERASURE_TARGET SET STATUS='pending',EVIDENCE_REFERENCE=?,LAST_CODE='HISTORICAL_FILES_QUEUED',UPDATED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=? AND TARGET='historical_files'`, "reviewed-file-plan:"+plan.Evidence, plan.RequestID); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE SET NEXT_ATTEMPT_AT=UTC_TIMESTAMP(),LAST_CODE='',UPDATED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, plan.RequestID); err != nil {
		return err
	}
	return tx.Commit()
}

func rejectReferencedHistoricalFile(tx *sqlx.Tx, schema erasureSchema, raw string) error {
	queries := []string{`SELECT COUNT(*) FROM ALUMNI_UPLOAD_OWNER WHERE RIGHT(URL_PATH,LENGTH(?))=?`}
	for _, column := range []string{"USR_PHOTO", "USR_BIZ_CARD", "USR_THUMNAIL"} {
		if schema.has("WEO_MEMBER", column) {
			queries = append(queries, "SELECT COUNT(*) FROM WEO_MEMBER WHERE RIGHT(`"+column+"`,LENGTH(?))=?")
		}
	}
	if schema["WEO_FILES"] != nil {
		queries = append(queries, `SELECT COUNT(*) FROM WEO_FILES WHERE F_JOIN_SEQ<>0 AND RIGHT(CONCAT(FILE_PATH,'/',FILE_NAME),LENGTH(?))=?`)
	}
	for _, query := range queries {
		var count int
		if err := tx.Get(&count, query, raw, raw); err != nil {
			return err
		}
		if count > 0 {
			return &model.ErasureBlocked{Code: "HISTORICAL_FILE_STILL_REFERENCED"}
		}
	}
	// Includes raw and base64-encoded post bodies; never delete an image still
	// embedded by another account even if its attachment row was lost.
	for _, column := range []string{"CONTENTS", "CONTENTS_MD", "THUMBNAIL_URL", "FILES", "RE_FILES"} {
		if !schema.has("WEO_BOARDBBS", column) {
			continue
		}
		var count int
		query := "SELECT COUNT(*) FROM WEO_BOARDBBS WHERE INSTR(COALESCE(`" + column + "`,''),?)>0 OR INSTR(COALESCE(FROM_BASE64(`" + column + "`),''),?)>0"
		if err := tx.Get(&count, query, raw, raw); err != nil {
			return err
		}
		if count > 0 {
			return &model.ErasureBlocked{Code: "HISTORICAL_FILE_STILL_REFERENCED"}
		}
	}
	return nil
}
