package handler

import (
	"context"
	"encoding/json"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"net/http/httptest"
	"strings"
	"testing"
)

type archiveStub struct{ calls int }

func (s *archiveStub) List(context.Context, int64) ([]repository.DonationArchiveMetadata, error) {
	s.calls++
	return []repository.DonationArchiveMetadata{}, nil
}
func (s *archiveStub) Read(_ context.Context, _ int64, _ int, _ string, _ func([]byte) ([]byte, error)) (json.RawMessage, error) {
	s.calls++
	return json.RawMessage(`{"donorName":"synthetic"}`), nil
}
func TestDonationArchiveAuthorizationAndPurpose(t *testing.T) {
	for _, tc := range []struct {
		name, role, purpose string
		want                int
	}{{"anonymous", "", "accounting_review", 401}, {"operator", "operator", "accounting_review", 403}, {"root_invalid", "root", "free text", 400}, {"root", "root", "accounting_review", 200}} {
		t.Run(tc.name, func(t *testing.T) {
			store := &archiveStub{}
			h := DonationArchiveHandler{Store: store, Open: func(b []byte) ([]byte, error) { return b, nil }}
			router := chi.NewRouter()
			router.Post("/{id}/read", h.Read)
			req := httptest.NewRequest("POST", "/7/read", strings.NewReader(`{"purpose":"`+tc.purpose+`"}`))
			if tc.role != "" {
				role := model.AdminRole(tc.role)
				req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42, AdminRole: &role}))
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("cacheable response")
			}
			if tc.want != 200 && store.calls != 0 {
				t.Fatal("unauthorized/invalid request reached store")
			}
		})
	}
}
