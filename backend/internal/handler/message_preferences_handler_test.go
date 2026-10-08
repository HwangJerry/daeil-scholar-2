// message_preferences_handler_test.go — Explicit boolean validation and authenticated account ownership.
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

type preferenceStore struct {
	allowed bool
	owner   int
	err     error
}

func (s *preferenceStore) Get(owner int) (*model.MessagePreferences, error) {
	s.owner = owner
	return &model.MessagePreferences{MessageAllowed: s.allowed}, s.err
}
func (s *preferenceStore) Save(owner int, p model.MessagePreferences) error {
	s.owner = owner
	if s.err != nil {
		return s.err
	}
	s.allowed = p.MessageAllowed
	return nil
}

func TestMessagePreferencesRoundTripUsesAuthenticatedAccount(t *testing.T) {
	store := &preferenceStore{allowed: true}
	h := &MessagePreferencesHandler{Service: &service.MessagePreferencesService{Store: store}}
	for _, flag := range []string{"false", "true"} {
		req := httptest.NewRequest(http.MethodPut, "/api/profile/message-preferences", strings.NewReader(`{"messageAllowed":`+flag+`}`))
		req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
		w := httptest.NewRecorder()
		h.Put(w, req)
		if w.Code != 200 || store.owner != 42 {
			t.Fatalf("PUT: %d owner=%d", w.Code, store.owner)
		}
		get := httptest.NewRequest(http.MethodGet, "/api/profile/message-preferences", nil)
		get = get.WithContext(middleware.SetAuthUser(get.Context(), &model.AuthUser{USRSeq: 42}))
		w = httptest.NewRecorder()
		h.Get(w, get)
		if w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"messageAllowed":`+flag+`}` {
			t.Fatalf("GET: %d %s", w.Code, w.Body.String())
		}
	}
}

func TestMessagePreferencesRejectsAmbiguousBodiesAndAnonymousUsers(t *testing.T) {
	store := &preferenceStore{allowed: true}
	h := &MessagePreferencesHandler{Service: &service.MessagePreferencesService{Store: store}}
	for _, body := range []string{`{}`, `{"messageAllowed":null}`, `{"messageAllowed":"false"}`, `{"messageAllowed":false,"usrSeq":7}`, `{"messageAllowed":false} {}`} {
		req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
		req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
		w := httptest.NewRecorder()
		h.Put(w, req)
		if w.Code != 400 || !store.allowed {
			t.Fatalf("body=%s: %d", body, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.Get(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != 401 {
		t.Fatalf("anonymous GET: %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.Put(w, httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"messageAllowed":false}`)))
	if w.Code != 401 {
		t.Fatalf("anonymous PUT: %d", w.Code)
	}
	store.err = errors.New("db down")
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"messageAllowed":false}`))
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	w = httptest.NewRecorder()
	h.Put(w, req)
	if w.Code != 500 || !store.allowed {
		t.Fatal("failed write was reported as success")
	}
}
