// erasure_receipt_work_test.go — Completion requires delivered results and actual contact cleanup.
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"testing"
)

type receiptWorkStub struct {
	deletionStoreStub
	calls int
}

func (s *receiptWorkStub) ResolveReceiptWork(int64, int, model.AccountDeletionResolution) error {
	s.calls++
	return nil
}
func TestReceiptContactWorkRequiresIndependentAttestations(t *testing.T) {
	store := &receiptWorkStub{}
	svc := &AccountDeletionRequestService{Store: store}
	request := model.AccountDeletionResolution{Action: "receipt_work", ReceiptWorkStatus: "active", OriginalStorage: "separate_excel", EvidenceReference: "synthetic-work-17"}
	if err := svc.Resolve(1, 7, request); err == nil {
		t.Fatal("unsecured original accepted")
	}
	request.ContactSecured = true
	if err := svc.Resolve(1, 7, request); err != nil {
		t.Fatal(err)
	}
	request.ReceiptWorkStatus = "completed"
	if err := svc.Resolve(1, 7, request); err == nil {
		t.Fatal("undelivered work accepted")
	}
	request.ResultNotified = true
	if err := svc.Resolve(1, 7, request); err == nil {
		t.Fatal("contact cleanup skipped")
	}
	request.ContactErased = true
	if err := svc.Resolve(1, 7, request); err != nil {
		t.Fatal(err)
	}
	if store.calls != 2 {
		t.Fatal("invalid transition reached storage")
	}
	request.EvidenceReference = ""
	if err := svc.Resolve(1, 7, request); err == nil {
		t.Fatal("missing evidence accepted")
	}
}
