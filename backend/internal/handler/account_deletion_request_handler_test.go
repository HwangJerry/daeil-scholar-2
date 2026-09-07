package handler

import (
	"errors"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type deletionHandlerStore struct {
	hash string
	fail bool
}

func (s *deletionHandlerStore) Create(_ int, hash string) (model.AccountDeletionReceipt, error) {
	s.hash = hash
	if s.fail {
		return model.AccountDeletionReceipt{}, errors.New("storage failure")
	}
	return model.AccountDeletionReceipt{ID: 17, Status: "pending"}, nil
}
func (*deletionHandlerStore) Receipt(string) (model.AccountDeletionReceipt, error) {
	return model.AccountDeletionReceipt{ID: 17, Status: "pending"}, nil
}
func (*deletionHandlerStore) List(string, int64) ([]model.AccountDeletionQueueItem, error) {
	return nil, nil
}
func (*deletionHandlerStore) Start(int64, int) error                                     { return nil }
func (*deletionHandlerStore) Verify(int64) ([]model.AccountDeletionFootprint, error)     { return nil, nil }
func (*deletionHandlerStore) Complete(int64, int, model.AccountDeletionResolution) error { return nil }

type deletionSessionStub struct{ calls int }

func (s *deletionSessionStub) LogoutAll(http.ResponseWriter, int) error {
	s.calls++
	return errors.New("temporary cache failure")
}

func TestDeletionRequestRequiresAuthentication(t *testing.T) {
	h := &AccountDeletionRequestHandler{}
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/auth/account/deletion-requests", nil))
	if w.Code != 401 {
		t.Fatalf("status=%d", w.Code)
	}
	for _, endpoint := range []http.HandlerFunc{h.List, h.Verify, h.Resolve} {
		w = httptest.NewRecorder()
		middleware.AdminAuthMiddleware(endpoint).ServeHTTP(w, authRequest(http.MethodGet, "/api/admin/account-deletions", nil))
		if w.Code != 403 {
			t.Fatalf("ordinary member can access deletion operations: %d", w.Code)
		}
	}
}

func TestPausedDeletionDoesNotDisableAccountOrLogout(t *testing.T) {
	store := &deletionHandlerStore{}
	session := &deletionSessionStub{}
	h := &AccountDeletionRequestHandler{RequestsDisabled: true, Service: &service.AccountDeletionRequestService{Store: store}, Auth: session}
	w := httptest.NewRecorder()
	h.Create(w, authRequest(http.MethodPost, "/api/auth/account/deletion-requests", nil))
	if w.Code != 503 || store.hash != "" || session.calls != 0 {
		t.Fatal("paused request mutated account", w.Code)
	}
}

func TestDeletionAcceptedDespitePostCommitSessionCleanupFailure(t *testing.T) {
	store := &deletionHandlerStore{}
	session := &deletionSessionStub{}
	h := &AccountDeletionRequestHandler{Service: &service.AccountDeletionRequestService{Store: store}, Auth: session}
	token := strings.Repeat("a", 64)
	w := httptest.NewRecorder()
	h.Create(w, authRequest(http.MethodPost, "/api/auth/account/deletion-requests", []byte(`{"receiptToken":"`+token+`"}`)))
	if w.Code != 202 || !strings.Contains(w.Body.String(), `"sessionCleanupPending":true`) || session.calls != 1 || store.hash == token {
		t.Fatalf("accepted request lost: %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("receipt can be cached")
	}
	store.fail = true
	w = httptest.NewRecorder()
	h.Create(w, authRequest(http.MethodPost, "/api/auth/account/deletion-requests", nil))
	if w.Code != 500 || session.calls != 1 {
		t.Fatal("session cleared before durable request")
	}
}
