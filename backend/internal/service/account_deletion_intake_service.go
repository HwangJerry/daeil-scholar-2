// account_deletion_intake_service.go — Operator intake of deletion requests received without the app.
package service

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/dflh-saf/backend/internal/model"
	"strings"
	"unicode/utf8"
)

const intakeEvidenceMaxRunes = 200

// CreateOnBehalf issues fresh receipt and cancel secrets for the member; they are
// returned once so the operator can send them in the reply and are stored only as hashes.
func (s *AccountDeletionRequestService) CreateOnBehalf(user, operator int, evidence string) (model.AccountDeletionReceipt, string, string, error) {
	evidence = strings.TrimSpace(evidence)
	if user <= 0 || operator <= 0 {
		return model.AccountDeletionReceipt{}, "", "", &model.ValidationError{Msg: "회원 번호를 확인해주세요."}
	}
	if user == operator {
		return model.AccountDeletionReceipt{}, "", "", &model.ValidationError{Msg: "본인 계정은 앱에서 탈퇴를 요청해주세요."}
	}
	if evidence == "" || utf8.RuneCountInString(evidence) > intakeEvidenceMaxRunes {
		return model.AccountDeletionReceipt{}, "", "", &model.ValidationError{Msg: "본인 확인 근거를 200자 이내로 입력해주세요."}
	}
	store, ok := s.Store.(interface {
		CreateOnBehalf(int, int, string, string, string) (model.AccountDeletionReceipt, error)
	})
	if !ok {
		return model.AccountDeletionReceipt{}, "", "", &model.ValidationError{Msg: "대신 접수 기능이 준비되지 않았습니다."}
	}
	receiptToken, err := newDeletionToken()
	if err != nil {
		return model.AccountDeletionReceipt{}, "", "", err
	}
	cancelToken, err := newDeletionToken()
	if err != nil {
		return model.AccountDeletionReceipt{}, "", "", err
	}
	receiptHash, _ := deletionTokenHash(receiptToken)
	cancelHash, _ := deletionTokenHash(cancelToken)
	receipt, err := store.CreateOnBehalf(user, operator, receiptHash, cancelHash, evidence)
	if err != nil {
		return model.AccountDeletionReceipt{}, "", "", err
	}
	return receipt, receiptToken, cancelToken, nil
}

func newDeletionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
