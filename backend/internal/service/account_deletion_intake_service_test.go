package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"strings"
	"testing"
)

type intakeStore struct {
	user, operator                  int
	receiptHash, cancelHash, reason string
	calls                           int
}

func (*intakeStore) Create(int, string) (model.AccountDeletionReceipt, error) {
	return model.AccountDeletionReceipt{}, nil
}
func (*intakeStore) Receipt(string) (model.AccountDeletionReceipt, error) {
	return model.AccountDeletionReceipt{}, nil
}
func (*intakeStore) List(string, int64) ([]model.AccountDeletionQueueItem, error) { return nil, nil }
func (*intakeStore) Start(int64, int) error                                       { return nil }
func (*intakeStore) Verify(int64) ([]model.AccountDeletionFootprint, error)       { return nil, nil }
func (*intakeStore) Complete(int64, int, model.AccountDeletionResolution) error   { return nil }
func (s *intakeStore) CreateOnBehalf(user, operator int, receiptHash, cancelHash, reason string) (model.AccountDeletionReceipt, error) {
	s.calls++
	s.user, s.operator, s.receiptHash, s.cancelHash, s.reason = user, operator, receiptHash, cancelHash, reason
	return model.AccountDeletionReceipt{ID: 3, Status: "pending"}, nil
}

func hashOf(t *testing.T, token string) string {
	raw, err := hex.DecodeString(token)
	if err != nil || len(raw) != 32 {
		t.Fatalf("token is not 32 random bytes")
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func TestCreateOnBehalfIssuesDistinctSecretsAndStoresOnlyHashes(t *testing.T) {
	store := &intakeStore{}
	service := &AccountDeletionRequestService{Store: store}
	receipt, receiptToken, cancelToken, err := service.CreateOnBehalf(42, 7, "  가입 이메일 요청 확인 ")
	if err != nil || receipt.ID != 3 {
		t.Fatal(receipt, err)
	}
	if receiptToken == cancelToken {
		t.Fatal("receipt lookup and cancel authority must differ")
	}
	if store.user != 42 || store.operator != 7 || store.reason != "가입 이메일 요청 확인" {
		t.Fatal("wrong intake", store.user, store.operator, store.reason)
	}
	if store.receiptHash != hashOf(t, receiptToken) || store.cancelHash != hashOf(t, cancelToken) {
		t.Fatal("store received something other than the token hashes")
	}
}

func TestCreateOnBehalfRejectsSelfMissingOrLongEvidence(t *testing.T) {
	store := &intakeStore{}
	service := &AccountDeletionRequestService{Store: store}
	for _, c := range []struct {
		user, operator int
		evidence       string
	}{{7, 7, "확인"}, {0, 7, "확인"}, {42, 7, "   "}, {42, 7, strings.Repeat("가", 201)}} {
		_, _, _, err := service.CreateOnBehalf(c.user, c.operator, c.evidence)
		var invalid *model.ValidationError
		if !errors.As(err, &invalid) {
			t.Fatal("accepted invalid intake", c.user, c.operator, err)
		}
	}
	if store.calls != 0 {
		t.Fatal("invalid intake reached storage")
	}
}
