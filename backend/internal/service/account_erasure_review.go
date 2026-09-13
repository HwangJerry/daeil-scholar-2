// account_erasure_review.go — Operator preview and plan-bound approval of an account erasure.
package service

import (
	"regexp"

	"github.com/dflh-saf/backend/internal/model"
)

var erasurePlanDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)

type erasureReviewStore interface {
	PreviewErasure(int64) (model.ErasurePreview, error)
	ExpediteReviewed(int64, int, string) error
}

// Preview lists the records the automatic erasure would change and how.
func (s *AccountDeletionRequestService) Preview(id int64) (model.ErasurePreview, error) {
	if id <= 0 {
		return model.ErasurePreview{}, &model.ValidationError{Msg: "올바른 요청을 선택해주세요."}
	}
	store, ok := s.Store.(erasureReviewStore)
	if !ok {
		return model.ErasurePreview{}, &model.ValidationError{Msg: "처리 대상 검토 기능이 준비되지 않았습니다."}
	}
	return store.PreviewErasure(id)
}

// expediteReviewed refuses immediate processing unless the operator approved
// a specific reviewed plan; the store re-checks it against current records.
func (s *AccountDeletionRequestService) expediteReviewed(id int64, operator int, digest string) error {
	if !erasurePlanDigest.MatchString(digest) {
		return &model.ValidationError{Msg: "처리 대상 기록을 먼저 검토해주세요."}
	}
	store, ok := s.Store.(erasureReviewStore)
	if !ok {
		return &model.ValidationError{Msg: "처리 대상 검토 기능이 준비되지 않았습니다."}
	}
	return store.ExpediteReviewed(id, operator, digest)
}
