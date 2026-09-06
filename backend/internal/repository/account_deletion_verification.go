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
            'REPORTER_SEQ','REPORTED_SEQ','BLOCKER_USR_SEQ','BLOCKED_USR_SEQ')
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
	footprint, err := deletionFootprint(tx, usrSeq)
	if err != nil {
		return err
	}
	if len(footprint) > 0 {
		return ErrDeletionIncomplete
	}
	var missingProof int
	err = tx.Get(&missingProof, `SELECT
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
	return tx.Commit()
}
