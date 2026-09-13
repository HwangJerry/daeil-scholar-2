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

func (s *AccountDeletionRequestService) CreateCancelable(user int, receiptToken, cancelToken string) (model.AccountDeletionReceipt, string, error) {
	if cancelToken == "" {
		return s.Create(user, receiptToken)
	} // Legacy clients remain receipt-only.
	receiptHash, err := deletionTokenHash(receiptToken)
	if err != nil {
		return model.AccountDeletionReceipt{}, "", err
	}
	cancelHash, err := deletionTokenHash(cancelToken)
	if err != nil {
		return model.AccountDeletionReceipt{}, "", err
	}
	if receiptHash == cancelHash {
		return model.AccountDeletionReceipt{}, "", &model.ValidationError{Msg: "조회 번호와 취소 인증값은 달라야 합니다."}
	}
	store, ok := s.Store.(interface {
		CreateCancelable(int, string, string) (model.AccountDeletionReceipt, error)
	})
	if !ok {
		return model.AccountDeletionReceipt{}, "", &model.ValidationError{Msg: "취소 가능한 접수 기능이 준비되지 않았습니다."}
	}
	receipt, err := store.CreateCancelable(user, receiptHash, cancelHash)
	return receipt, receiptToken, err
}
func (s *AccountDeletionRequestService) Cancel(receiptToken, cancelToken string) (model.AccountDeletionReceipt, error) {
	receiptHash, err := deletionTokenHash(receiptToken)
	if err != nil {
		return model.AccountDeletionReceipt{}, err
	}
	cancelHash, err := deletionTokenHash(cancelToken)
	if err != nil {
		return model.AccountDeletionReceipt{}, err
	}
	store, ok := s.Store.(interface {
		Cancel(string, string) (model.AccountDeletionReceipt, error)
	})
	if !ok {
		return model.AccountDeletionReceipt{}, &model.ValidationError{Msg: "취소 기능이 준비되지 않았습니다."}
	}
	return store.Cancel(receiptHash, cancelHash)
}

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
	if (status != "pending" && status != "processing" && status != "completed" && status != "cancelled") || before < 0 {
		return nil, &model.ValidationError{Msg: "올바른 삭제 요청 목록을 선택해주세요."}
	}
	return s.Store.List(status, before)
}

func (s *AccountDeletionRequestService) Resolve(id int64, operator int, request model.AccountDeletionResolution) error {
	if id <= 0 || operator <= 0 {
		return &model.ValidationError{Msg: "올바른 요청을 선택해주세요."}
	}
	if request.Action == "cancel_verified" {
		if strings.TrimSpace(request.EvidenceReference) == "" || len(request.EvidenceReference) > 500 {
			return &model.ValidationError{Msg: "본인 확인 근거를 입력해주세요."}
		}
		store, ok := s.Store.(interface {
			CancelVerified(int64, int, string) error
		})
		if !ok {
			return &model.ValidationError{Msg: "본인 확인 취소 기능이 준비되지 않았습니다."}
		}
		return store.CancelVerified(id, operator, request.EvidenceReference)
	}
	if request.Action == "target" {
		store, ok := s.Store.(interface {
			ReviewErasureTarget(int64, int, model.ErasureTarget) error
		})
		if !ok {
			return &model.ValidationError{Msg: "저장소 검토 기능이 준비되지 않았습니다."}
		}
		return store.ReviewErasureTarget(id, operator, model.ErasureTarget{Name: request.Target, Status: request.TargetStatus, Evidence: strings.TrimSpace(request.EvidenceReference)})
	}
	if request.Action == "expedite" {
		return s.expediteReviewed(id, operator, request.ReviewedPlanDigest)
	}
	if request.Action == "schedule" || request.Action == "retry_social" {
		store, ok := s.Store.(interface {
			ControlSchedule(int64, int, string) error
		})
		if !ok {
			return &model.ValidationError{Msg: "예약 탈퇴 저장소가 준비되지 않았습니다."}
		}
		return store.ControlSchedule(id, operator, request.Action)
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
	if request.Action == "receipt_work" {
		return s.resolveReceiptWork(id, operator, request)
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
