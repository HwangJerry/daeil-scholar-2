package handler

import (
	"encoding/json"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"net/http"
	"testing"

	"net/http/httptest"
)

type intakeHandlerStore struct {
	deletionHandlerStore
	user int
}

func (s *intakeHandlerStore) CreateOnBehalf(user, _ int, _, _, _ string) (model.AccountDeletionReceipt, error) {
	s.user = user
	return model.AccountDeletionReceipt{ID: 21, Status: "pending"}, nil
}

type intakeSessionStub struct{ logoutAll, revoked int }

func (s *intakeSessionStub) LogoutAll(http.ResponseWriter, int) error {
	s.logoutAll++
	return nil
}
func (s *intakeSessionStub) RevokeSessions(user int) error {
	s.revoked = user
	return nil
}

func TestProxyIntakeRevokesMemberSessionsWithoutTouchingOperatorCookies(t *testing.T) {
	store := &intakeHandlerStore{}
	session := &intakeSessionStub{}
	h := &AccountDeletionRequestHandler{Service: &service.AccountDeletionRequestService{Store: store}, Auth: session}
	w := httptest.NewRecorder()
	h.CreateOnBehalf(w, authRequest(http.MethodPost, "/api/admin/account-deletions", []byte(`{"userSeq":42,"evidenceReference":"가입 이메일 요청 확인"}`)))
	if w.Code != http.StatusCreated || store.user != 42 {
		t.Fatalf("status=%d user=%d", w.Code, store.user)
	}
	if session.revoked != 42 || session.logoutAll != 0 || w.Header().Get("Set-Cookie") != "" {
		t.Fatal("operator session touched or member sessions kept")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("one-time secrets cacheable")
	}
	var body struct {
		ReceiptToken string `json:"receiptToken"`
		CancelToken  string `json:"cancelToken"`
	}
	if json.Unmarshal(w.Body.Bytes(), &body) != nil || len(body.ReceiptToken) != 64 || len(body.CancelToken) != 64 || body.ReceiptToken == body.CancelToken {
		t.Fatal("secrets not returned once", w.Body.String())
	}
}

func TestProxyIntakePausedLeavesMemberUntouched(t *testing.T) {
	store := &intakeHandlerStore{}
	session := &intakeSessionStub{}
	for _, h := range []*AccountDeletionRequestHandler{
		{RequestsDisabled: true, Service: &service.AccountDeletionRequestService{Store: store}, Auth: session},
		{TestUserSeq: 5, Service: &service.AccountDeletionRequestService{Store: store}, Auth: session},
	} {
		w := httptest.NewRecorder()
		h.CreateOnBehalf(w, authRequest(http.MethodPost, "/api/admin/account-deletions", []byte(`{"userSeq":42,"evidenceReference":"확인"}`)))
		if w.Code != http.StatusServiceUnavailable || store.user != 0 || session.revoked != 0 {
			t.Fatal("paused intake mutated member", w.Code)
		}
	}
}
