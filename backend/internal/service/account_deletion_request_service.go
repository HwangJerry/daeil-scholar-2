// account_deletion_request_service.go — Receipt secrets and manual erasure workflow validation.
package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/dflh-saf/backend/internal/model"
	"strings"
	"time"
	"unicode/utf8"
)

type AccountDeletionRequestStore interface {
	Create(int, string) (model.AccountDeletionReceipt, error)
	Receipt(string) (model.AccountDeletionReceipt, error)
	List(string, int64) ([]model.AccountDeletionQueueItem, error)
	Start(int64, int) error
	Verify(int64) ([]model.AccountDeletionFootprint, error)
	Complete(int64, int, model.AccountDeletionResolution) error
}
type AccountDeletionRequestService struct{ Store AccountDeletionRequestStore }

func (s *AccountDeletionRequestService) Verify(id int64) ([]model.AccountDeletionFootprint, error) {
	return s.Store.Verify(id)
}

func deletionTokenHash(token string) (string, error) {
	bytes, err := hex.DecodeString(token)
	if err != nil || len(bytes) != 32 {
		return "", &model.ValidationError{Msg: "삭제 요청 확인번호를 확인해주세요."}
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:]), nil
}

func (s *AccountDeletionRequestService) Create(usrSeq int, token string) (model.AccountDeletionReceipt, string, error) {
	if token == "" {
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err != nil {
			return model.AccountDeletionReceipt{}, "", err
		}
		token = hex.EncodeToString(bytes)
	}
	hash, err := deletionTokenHash(token)
	if err != nil {
		return model.AccountDeletionReceipt{}, "", err
	}
	receipt, err := s.Store.Create(usrSeq, hash)
	return receipt, token, err
}

func (s *AccountDeletionRequestService) Receipt(token string) (model.AccountDeletionReceipt, error) {
	hash, err := deletionTokenHash(token)
	if err != nil {
		return model.AccountDeletionReceipt{}, err
	}
	return s.Store.Receipt(hash)
}

func (s *AccountDeletionRequestService) List(status string, before int64) ([]model.AccountDeletionQueueItem, error) {
	if status == "" {
		status = "pending"
	}
	if (status != "pending" && status != "processing" && status != "completed") || before < 0 {
		return nil, &model.ValidationError{Msg: "올바른 삭제 요청 목록을 선택해주세요."}
	}
	return s.Store.List(status, before)
}

func (s *AccountDeletionRequestService) Resolve(id int64, operator int, request model.AccountDeletionResolution) error {
	if id <= 0 || operator <= 0 {
		return &model.ValidationError{Msg: "올바른 요청을 선택해주세요."}
	}
	if request.Action == "automatic" || request.Action == "manual" {
		store, ok := s.Store.(interface {
			SetErasureMode(int64, int, string) error
		})
		if !ok {
			return &model.ValidationError{Msg: "자동 처리 저장소가 준비되지 않았습니다."}
		}
		return store.SetErasureMode(id, operator, request.Action)
	}
	if request.Action == "start" {
		return s.Store.Start(id, operator)
	}
	request.EvidenceReference = strings.TrimSpace(request.EvidenceReference)
	request.RetainedRecords = strings.TrimSpace(request.RetainedRecords)
	if request.Action != "complete" || !request.ResultNotified || !request.FilesErased || !request.BackupsErased || !request.ExternalDataErased || !request.OtherIdentifiersChecked ||
		request.EvidenceReference == "" || request.RetainedRecords == "" || utf8.RuneCountInString(request.EvidenceReference) > 1000 || utf8.RuneCountInString(request.RetainedRecords) > 1000 {
		return &model.ValidationError{Msg: "실제 삭제 작업과 증빙을 모두 확인해주세요. 보존 기록이 없으면 '없음'을 입력하세요."}
	}
	if request.RetainedRecords == "없음" {
		if request.RetentionUntil != "" {
			return &model.ValidationError{Msg: "보존 기록이 없으면 보존 종료일을 비워주세요."}
		}
	} else {
		until, err := time.Parse("2006-01-02", request.RetentionUntil)
		if err != nil || !until.After(time.Now().UTC()) {
			return &model.ValidationError{Msg: "보존 근거·항목과 미래의 보존 종료일이 필요합니다."}
		}
	}
	return s.Store.Complete(id, operator, request)
}
