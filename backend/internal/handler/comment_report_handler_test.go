package handler

import (
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommentReportingRequiresLogin(t *testing.T) {
	h := &CommentReportHandler{}
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/comment-reports", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestCommentModeratorEvidenceRequiresAdminRole(t *testing.T) {
	h := &CommentReportHandler{}
	w := httptest.NewRecorder()
	middleware.AdminAuthMiddleware(http.HandlerFunc(h.List)).ServeHTTP(w, authRequest(http.MethodGet, "/api/admin/comment-reports", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("ordinary user reached evidence queue: status=%d", w.Code)
	}
}

func TestCommentReportLimiterSpansCommentsAndDoesNotShareAccounts(t *testing.T) {
	limited := middleware.CommentReportRateLimiter()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		limited.ServeHTTP(w, authRequest("POST", "/api/feed/1/comments/11/reports", nil))
		if w.Code != 204 {
			t.Fatal(w.Code)
		}
	}
	w := httptest.NewRecorder()
	limited.ServeHTTP(w, authRequest("POST", "/api/feed/2/comments/12/reports", nil))
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal("account bucket not enforced")
	}
	r := authRequest("POST", "/api/feed/2/comments/12/reports", nil)
	r = r.WithContext(middleware.SetAuthUser(r.Context(), &model.AuthUser{USRSeq: 2}))
	w = httptest.NewRecorder()
	limited.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal("accounts shared rate budget")
	}
}
