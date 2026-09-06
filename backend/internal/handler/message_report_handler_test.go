package handler

import (
	"github.com/dflh-saf/backend/internal/middleware"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMessageReportingRequiresLogin(t *testing.T) {
	h := &MessageReportHandler{}
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/message-reports", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestModeratorEvidenceRequiresAdminRole(t *testing.T) {
	h := &MessageReportHandler{}
	w := httptest.NewRecorder()
	middleware.AdminAuthMiddleware(http.HandlerFunc(h.List)).ServeHTTP(w, authRequest(http.MethodGet, "/api/admin/message-reports", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("ordinary user reached evidence queue: status=%d", w.Code)
	}
}
