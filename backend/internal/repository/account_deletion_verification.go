// account_deletion_verification.go — Read-only database evidence before manual completion.
package repository

import (
	"database/sql"
	"fmt"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
	"regexp"
)

var deletionSQLIdentifier = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// This scan deliberately has no mutation mode. Deleting arbitrary legacy
// rows based only on a column name could destroy unrelated or legal records.
// The operator must additionally audit email/phone/provider IDs, indirect
// references, files and external systems using the documented runbook.
func deletionFootprint(tx *sqlx.Tx, usrSeq int) ([]model.AccountDeletionFootprint, error) {
	var columns []struct {
		Table  string `db:"TABLE_NAME"`
		Column string `db:"COLUMN_NAME"`
	}
	err := tx.Select(&columns, `SELECT c.TABLE_NAME, c.COLUMN_NAME FROM information_schema.COLUMNS c
        JOIN information_schema.TABLES t ON t.TABLE_SCHEMA = c.TABLE_SCHEMA AND t.TABLE_NAME = c.TABLE_NAME
        WHERE c.TABLE_SCHEMA = DATABASE() AND t.TABLE_TYPE = 'BASE TABLE'
        AND c.TABLE_NAME <> 'ALUMNI_ACCOUNT_DELETION_REQUEST'
        AND c.COLUMN_NAME IN ('USR_SEQ','ACCOUNT_ID','USER_ID','AM_SENDER_SEQ','AM_RECVR_SEQ',
            'REPORTER_SEQ','REPORTED_SEQ','BLOCKER_USR_SEQ','BLOCKED_USR_SEQ','VD_USR_SEQ','O_ACCOUNT_USR_SEQ')
        ORDER BY c.TABLE_NAME, c.COLUMN_NAME`)
	if err != nil {
		return nil, err
	}
	foundMember := false
	items := []model.AccountDeletionFootprint{}
	for _, col := range columns {
		if !deletionSQLIdentifier.MatchString(col.Table) || !deletionSQLIdentifier.MatchString(col.Column) {
			return nil, ErrDeletionIncomplete
		}
		if col.Table == "WEO_MEMBER" && col.Column == "USR_SEQ" {
			foundMember = true
		}
		query := fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE `%s` = ?", col.Table, col.Column)
		// Keep provider proof until after completion verification. Pending,
		// failed and unknown actions must still block completion.
		if col.Table == "ALUMNI_SOCIAL_REVOCATION_OUTBOX" {
			query += " AND STATUS <> 'DELIVERED'"
		}
		var count int64
		if err = tx.Get(&count, query, usrSeq); err != nil {
			return nil, err
		}
		if count > 0 {
			items = append(items, model.AccountDeletionFootprint{Table: col.Table, Column: col.Column, Count: count})
		}
	}
	if !foundMember {
		return nil, ErrDeletionIncomplete
	}
	return items, nil
}

func (r *AccountDeletionRequestRepository) Verify(id int64) ([]model.AccountDeletionFootprint, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var usrSeq int
	err = tx.Get(&usrSeq, `SELECT USR_SEQ FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID = ? AND STATUS <> 'completed'`, id)
	if err != nil {
		return nil, err
	}
	return deletionFootprint(tx, usrSeq)
}

func (r *AccountDeletionRequestRepository) Complete(id int64, operator int, evidence model.AccountDeletionResolution) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var usrSeq int
	if err = tx.Get(&usrSeq, `SELECT USR_SEQ FROM ALUMNI_ACCOUNT_DELETION_REQUEST
        WHERE REQUEST_ID = ? AND STATUS = 'processing' AND USR_SEQ <> ? FOR UPDATE`, id, operator); err != nil {
		return err
	}
	var automation struct {
		Mode     string `db:"MODE"`
		Stage    string `db:"STAGE"`
		Evidence string `db:"EXTERNAL_EVIDENCE"`
	}
	if err = tx.Get(&automation, `SELECT MODE,STAGE,EXTERNAL_EVIDENCE FROM ALUMNI_ACCOUNT_ERASURE WHERE REQUEST_ID=? FOR UPDATE`, id); err != nil {
		return err
	}
	if operator == 0 {
		verified, e := verifiedErasureTargets(tx, id)
		if e != nil {
			return e
		}
		if !verified {
			return ErrDeletionIncomplete
		}
		var pendingFiles int
		if err = tx.Get(&pendingFiles, `SELECT COUNT(*) FROM ALUMNI_ERASURE_FILE WHERE REQUEST_ID=?`, id); err != nil {
			return err
		}
		if automation.Mode != "automatic" || automation.Stage != "database_erased" || automation.Evidence == "" || pendingFiles > 0 {
			return ErrDeletionIncomplete
		}
	} else if automation.Mode != "manual" {
		return ErrDeletionIncomplete
	}
	footprint, err := deletionFootprint(tx, usrSeq)
	if err != nil {
		return err
	}
	if len(footprint) > 0 {
		return ErrDeletionIncomplete
	}
	if err = verifyDeletionProviders(tx, id, usrSeq); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX WHERE USR_SEQ = ? AND STATUS = 'DELIVERED'`, usrSeq); err != nil {
		return err
	}
	var retentionUntil sql.NullString
	if evidence.RetentionUntil != "" {
		retentionUntil = sql.NullString{String: evidence.RetentionUntil, Valid: true}
	}
	_, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_DELETION_REQUEST SET STATUS = 'completed',
        USR_SEQ = NULL, OPERATOR_SEQ = ?, COMPLETED_AT = UTC_TIMESTAMP(),
        EVIDENCE_REFERENCE = ?, RETAINED_RECORDS = ?, RETENTION_UNTIL = ? WHERE REQUEST_ID = ?`,
		operator, evidence.EvidenceReference, evidence.RetainedRecords, retentionUntil, id)
	if err != nil {
		return err
	}
	if operator != 0 {
		if _, err = tx.Exec(`UPDATE ALUMNI_ERASURE_TARGET SET STATUS='complete',EVIDENCE_REFERENCE='manual-completion: see request evidence',LAST_CODE='',UPDATED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=? AND STATUS NOT IN ('complete','not_applicable')`, id); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`DELETE FROM ALUMNI_ERASURE_CONTEXT WHERE REQUEST_ID=?`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM ALUMNI_ERASURE_FILE WHERE REQUEST_ID=?`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE SET STAGE='completed',LAST_CODE='',EXTERNAL_EVIDENCE='',UPDATED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func verifyDeletionProviders(tx *sqlx.Tx, id int64, usrSeq int) error {
	var missingProof int
	err := tx.Get(&missingProof, `SELECT
        (APPLE_REQUIRED = 1 AND NOT EXISTS (SELECT 1 FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX
            WHERE USR_SEQ = ? AND PROVIDER = 'AP' AND ACTION = 'ACCOUNT_DELETE' AND STATUS = 'DELIVERED'
            AND CREATED_AT >= ALUMNI_ACCOUNT_DELETION_REQUEST.REQUESTED_AT)) +
        (KAKAO_REQUIRED = 1 AND NOT EXISTS (SELECT 1 FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX
            WHERE USR_SEQ = ? AND PROVIDER = 'KT' AND ACTION = 'ACCOUNT_DELETE' AND STATUS = 'DELIVERED'
            AND CREATED_AT >= ALUMNI_ACCOUNT_DELETION_REQUEST.REQUESTED_AT))
        FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID = ?`, usrSeq, usrSeq, id)
	if err != nil {
		return err
	}
	if missingProof != 0 {
		return ErrDeletionIncomplete
	}
	return nil
}
