// erasure_receipt_work.go — Existing operator workflow controls limited-purpose contact retention.
package repository

import (
	"database/sql"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func (r *AccountDeletionRequestRepository) ReceiptWork(id int64) (model.ErasureReceiptWork, error) {
	var work model.ErasureReceiptWork
	err := r.DB.Get(&work, `SELECT STATUS,ORIGINAL_STORAGE,EVIDENCE_REFERENCE,UPDATED_AT,COMPLETED_AT FROM ALUMNI_ERASURE_RECEIPT_WORK WHERE REQUEST_ID=?`, id)
	work.UpdatedAt = deletionTimeUTC(work.UpdatedAt)
	if work.CompletedAt != nil {
		v := deletionTimeUTC(*work.CompletedAt)
		work.CompletedAt = &v
	}
	return work, err
}
func (r *AccountDeletionRequestRepository) ResolveReceiptWork(id int64, operator int, request model.AccountDeletionResolution) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var user int
	if err = tx.Get(&user, `SELECT USR_SEQ FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND STATUS<>'completed' AND USR_SEQ<>? FOR UPDATE`, id, operator); err != nil {
		return err
	}
	var current string
	if err = tx.Get(&current, `SELECT STATUS FROM ALUMNI_ERASURE_RECEIPT_WORK WHERE REQUEST_ID=? FOR UPDATE`, id); err != nil {
		return err
	}
	next := request.ReceiptWorkStatus
	// Active work cannot be dismissed as unnecessary; closing it requires the
	// service's delivery + contact-erasure attestations. Closed work cannot reopen.
	if current == "completed" || (next == "not_required" && current != "unreviewed") || (next == "active" && current != "unreviewed") || (next == "completed" && current != "active") {
		return &model.ValidationError{Msg: "현재 영수증 업무 상태에서 사용할 수 없는 변경입니다."}
	}
	if next != "not_required" && next != "active" && next != "completed" {
		return sql.ErrNoRows
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ERASURE_RECEIPT_WORK SET STATUS=?,ORIGINAL_STORAGE=CASE WHEN ?='active' THEN ? ELSE ORIGINAL_STORAGE END,EVIDENCE_REFERENCE=?,OPERATOR_SEQ=?,UPDATED_AT=UTC_TIMESTAMP(),COMPLETED_AT=CASE WHEN ?='completed' THEN UTC_TIMESTAMP() ELSE NULL END WHERE REQUEST_ID=?`, next, next, request.OriginalStorage, request.EvidenceReference, operator, next, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE SET NEXT_ATTEMPT_AT=UTC_TIMESTAMP(),LAST_CODE='' WHERE REQUEST_ID=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
func reviewReceiptWork(tx *sqlx.Tx, w model.ErasureWork, hasDonations bool) error {
	var status string
	if err := tx.Get(&status, `SELECT STATUS FROM ALUMNI_ERASURE_RECEIPT_WORK WHERE REQUEST_ID=? FOR UPDATE`, w.RequestID); err != nil {
		return err
	}
	if status == "unreviewed" {
		if hasDonations {
			return &model.ErasureBlocked{Code: "RECEIPT_CONTACT_REVIEW_REQUIRED"}
		}
		_, err := tx.Exec(`UPDATE ALUMNI_ERASURE_RECEIPT_WORK SET STATUS='not_required',EVIDENCE_REFERENCE='verified-no-linked-app-donations',UPDATED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, w.RequestID)
		return err
	}
	return nil
}
func receiptWorkFinished(tx *sqlx.Tx, id int64) (bool, error) {
	var count int
	err := tx.Get(&count, `SELECT COUNT(*) FROM ALUMNI_ERASURE_RECEIPT_WORK WHERE REQUEST_ID=? AND STATUS IN ('not_required','completed')`, id)
	return count == 1, err
}
