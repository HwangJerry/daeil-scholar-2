// account_deletion_intake.go — Requests registered by a root operator for members who asked outside the app.
package repository

import (
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// CreateOnBehalf creates the same cancellable request as the app and records who
// verified the member's identity, atomically with the request itself.
func (r *AccountDeletionRequestRepository) CreateOnBehalf(usrSeq, operator int, receiptHash, cancelHash, evidence string) (model.AccountDeletionReceipt, error) {
	if operator <= 0 || usrSeq == operator {
		return model.AccountDeletionReceipt{}, &model.ValidationError{Msg: "본인 계정은 앱에서 탈퇴를 요청해주세요."}
	}
	if r.TestUserSeq > 0 && usrSeq != r.TestUserSeq {
		return model.AccountDeletionReceipt{}, &model.ValidationError{Msg: "현재 테스트 회원만 처리할 수 있습니다."}
	}
	receipt, err := r.createReceipt(usrSeq, receiptHash, cancelHash, func(tx *sqlx.Tx, created model.AccountDeletionReceipt) error {
		_, err := tx.Exec(`INSERT INTO ALUMNI_ACCOUNT_DELETION_INTAKE (REQUEST_ID,OPERATOR_SEQ,EVIDENCE_REFERENCE,CREATED_AT) VALUES (?,?,?,UTC_TIMESTAMP())`, created.ID, operator, evidence)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return model.AccountDeletionReceipt{}, &model.ValidationError{Msg: "회원 번호를 찾을 수 없습니다."}
	}
	return receipt, err
}
