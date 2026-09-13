package service

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

type erasureReviewStub struct {
	deletionStoreStub
	previewed  int64
	expedited  []interface{}
	controlled bool
}

func (s *erasureReviewStub) PreviewErasure(id int64) (model.ErasurePreview, error) {
	s.previewed = id
	return model.ErasurePreview{RequestID: id}, nil
}
func (s *erasureReviewStub) ExpediteReviewed(id int64, operator int, digest string) error {
	s.expedited = append(s.expedited, id, operator, digest)
	return nil
}
func (s *erasureReviewStub) ControlSchedule(int64, int, string) error {
	s.controlled = true
	return nil
}

func TestExpediteRequiresReviewedPlanDigest(t *testing.T) {
	store := &erasureReviewStub{}
	svc := &AccountDeletionRequestService{Store: store}
	for _, digest := range []string{"", "abc", strings.Repeat("g", 64), strings.Repeat("A", 64)} {
		err := svc.Resolve(7, 3, model.AccountDeletionResolution{Action: "expedite", ReviewedPlanDigest: digest})
		var invalid *model.ValidationError
		if !errors.As(err, &invalid) {
			t.Fatalf("digest %q: err = %v", digest, err)
		}
	}
	if len(store.expedited) != 0 || store.controlled {
		t.Fatal("unreviewed expedite reached the store")
	}
	digest := strings.Repeat("a", 64)
	if err := svc.Resolve(7, 3, model.AccountDeletionResolution{Action: "expedite", ReviewedPlanDigest: digest}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.expedited, []interface{}{int64(7), 3, digest}) || store.controlled {
		t.Fatalf("expedite = %v, bypass = %v", store.expedited, store.controlled)
	}
}

func TestErasurePreviewRejectsInvalidRequest(t *testing.T) {
	store := &erasureReviewStub{}
	svc := &AccountDeletionRequestService{Store: store}
	if _, err := svc.Preview(0); err == nil {
		t.Fatal("accepted invalid request id")
	}
	if preview, err := svc.Preview(7); err != nil || preview.RequestID != 7 || store.previewed != 7 {
		t.Fatalf("preview = %+v, err = %v", preview, err)
	}
}
