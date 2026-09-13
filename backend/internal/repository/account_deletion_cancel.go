package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
)

var ErrDeletionCancellationClosed = errors.New("account deletion cancellation closed")

func (r *AccountDeletionRequestRepository) Cancel(receiptHash, cancelHash string) (model.AccountDeletionReceipt, error) {
	var result model.AccountDeletionReceipt
	release, locked, err := r.ErasureLock(context.Background())
	if err != nil {
		return result, err
	}
	if !locked {
		return result, ErrDeletionCancellationClosed
	}
	defer release()
	tx, err := r.DB.Beginx()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	var row struct {
		ID       int64         `db:"REQUEST_ID"`
		User     sql.NullInt64 `db:"USR_SEQ"`
		Status   string        `db:"STATUS"`
		Original string        `db:"ORIGINAL_STATUS"`
		Blocked  bool          `db:"RESTORE_BLOCKED"`
		Started  sql.NullTime  `db:"STARTED_AT"`
	}
	err = tx.Get(&row, `SELECT d.REQUEST_ID,d.USR_SEQ,d.STATUS,c.ORIGINAL_STATUS,c.RESTORE_BLOCKED,c.STARTED_AT FROM ALUMNI_ACCOUNT_DELETION_REQUEST d JOIN ALUMNI_ERASURE_CANCELLATION c ON c.REQUEST_ID=d.REQUEST_ID WHERE d.RECEIPT_HASH=? AND c.CANCEL_HASH=? AND c.CANCEL_HASH<>'' FOR UPDATE`, receiptHash, cancelHash)
	if err != nil {
		return result, err
	}
	if row.Status == "cancelled" {
		tx.Rollback()
		return r.Receipt(receiptHash)
	}
	if row.Status != "pending" || row.Started.Valid || row.Blocked || !row.User.Valid {
		return result, ErrDeletionCancellationClosed
	}
	var current string
	if err = tx.Get(&current, `SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=? FOR UPDATE`, row.User.Int64); err != nil {
		return result, err
	}
	if current != "AAA" {
		return result, ErrDeletionCancellationClosed
	}
	var unsafe int
	if err = tx.Get(&unsafe, `SELECT COUNT(*) FROM ALUMNI_ERASURE_TARGET WHERE REQUEST_ID=? AND STATUS NOT IN ('pending','manual')`, row.ID); err != nil {
		return result, err
	}
	if unsafe > 0 {
		return result, ErrDeletionCancellationClosed
	}
	restore := row.Original
	switch restore {
	case "CCC", "BBB", "BAA":
	case "ZZZ":
		restore = "CCC"
	default:
		return result, ErrDeletionCancellationClosed
	}
	if _, err = tx.Exec(`UPDATE WEO_MEMBER SET USR_STATUS=? WHERE USR_SEQ=?`, restore, row.User.Int64); err != nil {
		return result, err
	}
	// Old sessions and administrator roles stay revoked. A fresh login is required.
	if _, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_DELETION_REQUEST SET STATUS='cancelled',USR_SEQ=NULL,COMPLETED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, row.ID); err != nil {
		return result, err
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE SET MODE='manual',STAGE='cancelled',LAST_CODE='',UPDATED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, row.ID); err != nil {
		return result, err
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ERASURE_CANCELLATION SET CANCELLED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, row.ID); err != nil {
		return result, err
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	return r.Receipt(receiptHash)
}

func (r *AccountDeletionRequestRepository) CancelVerified(id int64, operator int, evidence string) error {
	if operator <= 0 {
		return ErrDeletionCancellationClosed
	}
	if err := r.checkOperatorErasureScope(id, operator); err != nil {
		return err
	}
	var keys struct {
		Receipt string `db:"RECEIPT_HASH"`
		Cancel  string `db:"CANCEL_HASH"`
	}
	if err := r.DB.Get(&keys, `SELECT d.RECEIPT_HASH,c.CANCEL_HASH FROM ALUMNI_ACCOUNT_DELETION_REQUEST d JOIN ALUMNI_ERASURE_CANCELLATION c ON c.REQUEST_ID=d.REQUEST_ID WHERE d.REQUEST_ID=? AND d.STATUS='pending' AND c.CANCEL_HASH<>''`, id); err != nil {
		return err
	}
	// Record identity verification before the state transition; never store the secret in audit.
	if _, err := r.DB.Exec(`UPDATE ALUMNI_ACCOUNT_DELETION_REQUEST SET OPERATOR_SEQ=?,EVIDENCE_REFERENCE=? WHERE REQUEST_ID=? AND STATUS='pending'`, operator, evidence, id); err != nil {
		return err
	}
	_, err := r.Cancel(keys.Receipt, keys.Cancel)
	return err
}
