// erasure_receipt_work.go — Require actual receipt work and delivery before contact cleanup.
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"strings"
)

func (s *AccountDeletionRequestService) resolveReceiptWork(id int64, operator int, request model.AccountDeletionResolution) error {
	request.EvidenceReference = strings.TrimSpace(request.EvidenceReference)
	invalid := &model.ValidationError{Msg: "영수증 업무와 원본 보관·연락처 정리 증빙을 확인해주세요. 개인정보 원문은 입력하지 마세요."}
	if request.EvidenceReference == "" || len(request.EvidenceReference) > 200 {
		return invalid
	}
	switch request.ReceiptWorkStatus {
	case "active":
		if !request.ContactSecured || (request.OriginalStorage != "happy_nanum" && request.OriginalStorage != "separate_excel" && request.OriginalStorage != "both") {
			return invalid
		}
	case "completed":
		if !request.ResultNotified || !request.ContactErased {
			return invalid
		}
	case "not_required":
	default:
		return invalid
	}
	store, ok := s.Store.(interface {
		ResolveReceiptWork(int64, int, model.AccountDeletionResolution) error
	})
	if !ok {
		return invalid
	}
	return store.ResolveReceiptWork(id, operator, request)
}
