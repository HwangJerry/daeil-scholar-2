package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"strings"
	"testing"
)

type deletionStoreStub struct {
	hash      string
	completed bool
}

func (s *deletionStoreStub) Create(_ int, hash string) (model.AccountDeletionReceipt, error) {
	s.hash = hash
	return model.AccountDeletionReceipt{ID: 1, Status: "pending"}, nil
}
func (s *deletionStoreStub) Receipt(hash string) (model.AccountDeletionReceipt, error) {
	s.hash = hash
	return model.AccountDeletionReceipt{ID: 1}, nil
}
func (*deletionStoreStub) List(string, int64) ([]model.AccountDeletionQueueItem, error) {
	return nil, nil
}
func (*deletionStoreStub) Start(int64, int) error                                 { return nil }
func (*deletionStoreStub) Verify(int64) ([]model.AccountDeletionFootprint, error) { return nil, nil }
func (s *deletionStoreStub) Complete(int64, int, model.AccountDeletionResolution) error {
	s.completed = true
	return nil
}

func TestDeletionReceiptSecretNeverStoredInPlaintext(t *testing.T) {
	store := &deletionStoreStub{}
	service := &AccountDeletionRequestService{Store: store}
	_, token, err := service.Create(7, "")
	if err != nil || len(token) != 64 || store.hash == token || len(store.hash) != 64 {
		t.Fatalf("secret handling failed: %v", err)
	}
	hash := store.hash
	if _, err = service.Receipt(token); err != nil || store.hash != hash {
		t.Fatal("receipt lookup hash differs")
	}
	for _, invalid := range []string{"", "123", strings.Repeat("z", 64), strings.Repeat("a", 66)} {
		if _, err = service.Receipt(invalid); err == nil {
			t.Fatal("accepted invalid receipt")
		}
	}
}

func TestDeletionCompletionRequiresEveryEvidenceAndNotification(t *testing.T) {
	valid := model.AccountDeletionResolution{Action: "complete", EvidenceReference: "work-ticket-1", RetainedRecords: "없음", FilesErased: true, BackupsErased: true, ExternalDataErased: true, OtherIdentifiersChecked: true, ResultNotified: true}
	cases := []struct {
		name   string
		change func(*model.AccountDeletionResolution)
	}{
		{"files", func(r *model.AccountDeletionResolution) { r.FilesErased = false }},
		{"backups", func(r *model.AccountDeletionResolution) { r.BackupsErased = false }},
		{"external", func(r *model.AccountDeletionResolution) { r.ExternalDataErased = false }},
		{"indirect", func(r *model.AccountDeletionResolution) { r.OtherIdentifiersChecked = false }},
		{"notification", func(r *model.AccountDeletionResolution) { r.ResultNotified = false }},
		{"evidence", func(r *model.AccountDeletionResolution) { r.EvidenceReference = " " }},
		{"retention without date", func(r *model.AccountDeletionResolution) { r.RetainedRecords = "기부금 영수증" }},
		{"contradictory retention", func(r *model.AccountDeletionResolution) { r.RetentionUntil = "2099-01-01" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &deletionStoreStub{}
			service := &AccountDeletionRequestService{Store: store}
			input := valid
			tc.change(&input)
			if service.Resolve(1, 7, input) == nil || store.completed {
				t.Fatal("unverified completion accepted")
			}
		})
	}
	store := &deletionStoreStub{}
	service := &AccountDeletionRequestService{Store: store}
	if err := service.Resolve(1, 7, valid); err != nil || !store.completed {
		t.Fatalf("verified completion rejected: %v", err)
	}
}
