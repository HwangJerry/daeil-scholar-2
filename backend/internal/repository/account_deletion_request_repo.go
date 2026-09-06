// account_deletion_request_repo.go — Atomic receipt creation and manual processing queue.
package repository

import (
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
	"time"
)

var ErrDeletionIncomplete = errors.New("account erasure verification incomplete")
var ErrDeletionReceiptConflict = errors.New("existing deletion request uses a different receipt")

type AccountDeletionRequestRepository struct {
	DB                        *sqlx.DB
	DonationRetentionTemplate func(int, time.Time) (model.DonationRetentionDecision, error)
}

const deletionReceiptColumns = `REQUEST_ID, STATUS, REQUESTED_AT, TARGET_AT, DUE_AT,
    COMPLETED_AT, RETAINED_RECORDS, RETENTION_UNTIL`

func (r *AccountDeletionRequestRepository) Create(usrSeq int, receiptHash string) (model.AccountDeletionReceipt, error) {
	var result model.AccountDeletionReceipt
	var engine string
	if err := r.DB.Get(&engine, `SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'WEO_MEMBER'`); err != nil {
		return result, err
	}
	if engine != "InnoDB" {
		return result, errors.New("account deletion requires transactional member storage")
	}
	tx, err := r.DB.Beginx()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	var status string
	if err = tx.Get(&status, `SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ = ? FOR UPDATE`, usrSeq); err != nil {
		return result, err
	}
	var existingHash string
	err = tx.Get(&existingHash, `SELECT RECEIPT_HASH FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE USR_SEQ = ?`, usrSeq)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	if err == nil && existingHash != receiptHash {
		return result, ErrDeletionReceiptConflict
	}
	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.Exec(`INSERT INTO ALUMNI_ACCOUNT_DELETION_REQUEST
            (USR_SEQ, RECEIPT_HASH, REQUESTED_AT, TARGET_AT, DUE_AT, APPLE_REQUIRED, KAKAO_REQUIRED)
            VALUES (?, ?, UTC_TIMESTAMP(), DATE_ADD(UTC_TIMESTAMP(), INTERVAL 3 DAY), DATE_ADD(UTC_TIMESTAMP(), INTERVAL 10 DAY),
                EXISTS(SELECT 1 FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ = ? AND NMS_GATE = 'AP'),
                EXISTS(SELECT 1 FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ = ? AND NMS_GATE = 'KT'))`, usrSeq, receiptHash, usrSeq, usrSeq)
		if err != nil {
			return result, err
		}
	}
	// Disable access in the same transaction as the durable receipt. Erasure
	// is a separate manual step; no provider credential is discarded here.
	if _, err = tx.Exec(`UPDATE WEO_MEMBER SET USR_STATUS = 'AAA' WHERE USR_SEQ = ?`, usrSeq); err != nil {
		return result, err
	}
	if _, err = tx.Exec(`DELETE FROM ALUMNI_ADMIN_ROLE WHERE USR_SEQ = ?`, usrSeq); err != nil {
		return result, err
	}
	if err = tx.Get(&result, `SELECT `+deletionReceiptColumns+` FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE USR_SEQ = ?`, usrSeq); err != nil {
		return result, err
	}
	if _, err = tx.Exec(`INSERT IGNORE INTO ALUMNI_ACCOUNT_ERASURE (REQUEST_ID, MODE, STAGE, NEXT_ATTEMPT_AT, UPDATED_AT) VALUES (?, 'automatic', 'queued', UTC_TIMESTAMP(), UTC_TIMESTAMP())`, result.ID); err != nil {
		return result, err
	}
	normalizeDeletionReceiptTimes(&result)
	return result, tx.Commit()
}

func (r *AccountDeletionRequestRepository) Receipt(hash string) (model.AccountDeletionReceipt, error) {
	var receipt model.AccountDeletionReceipt
	err := r.DB.Get(&receipt, `SELECT `+deletionReceiptColumns+` FROM ALUMNI_ACCOUNT_DELETION_REQUEST
        WHERE RECEIPT_HASH = ? AND (COMPLETED_AT IS NULL OR COMPLETED_AT > DATE_SUB(UTC_TIMESTAMP(), INTERVAL 30 DAY))`, hash)
	if err == nil {
		normalizeDeletionReceiptTimes(&receipt)
	}
	return receipt, err
}

func (r *AccountDeletionRequestRepository) List(status string, before int64) ([]model.AccountDeletionQueueItem, error) {
	items := []model.AccountDeletionQueueItem{}
	err := r.DB.Select(&items, `SELECT `+deletionReceiptColumns+`, USR_SEQ, EVIDENCE_REFERENCE,
        COALESCE((SELECT MODE FROM ALUMNI_ACCOUNT_ERASURE e WHERE e.REQUEST_ID=ALUMNI_ACCOUNT_DELETION_REQUEST.REQUEST_ID),'manual') AS PROCESSING_MODE,
        COALESCE((SELECT STAGE FROM ALUMNI_ACCOUNT_ERASURE e WHERE e.REQUEST_ID=ALUMNI_ACCOUNT_DELETION_REQUEST.REQUEST_ID),'') AS AUTO_STAGE,
        COALESCE((SELECT LAST_CODE FROM ALUMNI_ACCOUNT_ERASURE e WHERE e.REQUEST_ID=ALUMNI_ACCOUNT_DELETION_REQUEST.REQUEST_ID),'') AS AUTO_CODE
        FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE STATUS = ? AND (? = 0 OR REQUEST_ID < ?)
        ORDER BY REQUEST_ID DESC LIMIT 50`, status, before, before)
	for i := range items {
		normalizeDeletionReceiptTimes(&items[i].AccountDeletionReceipt)
	}
	return items, err
}

func (r *AccountDeletionRequestRepository) Start(id int64, operator int) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var usrSeq int
	if err = tx.Get(&usrSeq, `SELECT USR_SEQ FROM ALUMNI_ACCOUNT_DELETION_REQUEST
        WHERE REQUEST_ID = ? AND STATUS = 'pending' AND USR_SEQ <> ? FOR UPDATE`, id, operator); err != nil {
		return err
	}
	// The existing worker performs provider revocation before manual removal
	// of the identity rows. Missing credentials remain a visible blocker.
	_, err = tx.Exec(`INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX
        (USR_SEQ, PROVIDER, ACTION, STATUS, NEXT_ATTEMPT_AT, CREATED_AT, UPDATED_AT)
        SELECT s.USR_SEQ, s.NMS_GATE, 'ACCOUNT_DELETE', 'PENDING', UTC_TIMESTAMP(), UTC_TIMESTAMP(), UTC_TIMESTAMP()
        FROM WEO_MEMBER_SOCIAL s WHERE s.USR_SEQ = ? AND s.NMS_GATE IN ('AP','KT')
        AND NOT EXISTS (SELECT 1 FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX o
            WHERE o.USR_SEQ = s.USR_SEQ AND o.PROVIDER = s.NMS_GATE AND o.ACTION = 'ACCOUNT_DELETE'
            AND o.CREATED_AT >= (SELECT REQUESTED_AT FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID = ?))`, usrSeq, id)
	if err != nil {
		return err
	}
	if operator > 0 {
		if _, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE SET MODE='manual' WHERE REQUEST_ID=?`, id); err != nil {
			return err
		}
	}
	_, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_DELETION_REQUEST SET STATUS = 'processing', OPERATOR_SEQ = ? WHERE REQUEST_ID = ?`, operator, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// This table stores UTC DATETIME values. The application's shared MySQL driver
// parses legacy DATETIME in Asia/Seoul, so reinterpret the wall clock here;
// calling t.UTC() would incorrectly shift the stored instant by nine hours.
func normalizeDeletionReceiptTimes(receipt *model.AccountDeletionReceipt) {
	asUTC := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	}
	receipt.RequestedAt = asUTC(receipt.RequestedAt)
	receipt.TargetAt = asUTC(receipt.TargetAt)
	receipt.DueAt = asUTC(receipt.DueAt)
	if receipt.CompletedAt != nil {
		normalized := asUTC(*receipt.CompletedAt)
		receipt.CompletedAt = &normalized
	}
	if receipt.RetentionUntil != nil {
		normalized := asUTC(*receipt.RetentionUntil)
		receipt.RetentionUntil = &normalized
	}
}
