// notification_handler_test.go — Unit tests for NotificationHandler HTTP endpoints
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// stubNotificationRepo is a minimal NotificationQuerier stub for handler tests.
type stubNotificationRepo struct {
	getByUserPage int
	getByUserSize int
}

func (s *stubNotificationRepo) Insert(usrSeq int, notiType, title, body string, refSeq *int) error {
	return nil
}

func (s *stubNotificationRepo) GetByUser(usrSeq, page, size int) ([]model.Notification, int, error) {
	s.getByUserPage = page
	s.getByUserSize = size
	return []model.Notification{}, 0, nil
}

func (s *stubNotificationRepo) GetUnreadCount(usrSeq int) (int, error) { return 0, nil }
func (s *stubNotificationRepo) MarkAsRead(anSeq, usrSeq int) error     { return nil }
func (s *stubNotificationRepo) MarkAllAsRead(usrSeq int) error         { return nil }

// stubAuthRepo is a minimal AuthQuerier stub for handler tests.
type stubAuthRepo struct{}

func (s *stubAuthRepo) GetMemberBySeq(usrSeq int) (*model.User, error) { return nil, nil }

func newTestNotificationHandler() (*NotificationHandler, *stubNotificationRepo) {
	notiRepo := &stubNotificationRepo{}
	emailCh := make(chan model.EmailMessage, 10)
	logger := zerolog.Nop()
	svc := service.NewNotificationService(notiRepo, &stubAuthRepo{}, emailCh, logger, "http://localhost")
	h := NewNotificationHandler(svc, logger)
	return h, notiRepo
}

func authContext(ctx context.Context) context.Context {
	return middleware.SetAuthUser(ctx, &model.AuthUser{
		USRSeq:    1,
		USRID:     "testuser",
		USRName:   "Test User",
		USRStatus: "BBB",
	})
}

func TestGetNotifications_NoAuth(t *testing.T) {
	h, _ := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	rr := httptest.NewRecorder()
	h.GetNotifications(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}

	var apiErr model.APIError
	if err := json.NewDecoder(rr.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if apiErr.Code != "UNAUTHORIZED" {
		t.Errorf("expected error code UNAUTHORIZED, got %s", apiErr.Code)
	}
}

func TestGetNotifications_DefaultPagination(t *testing.T) {
	h, repo := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	req = req.WithContext(authContext(req.Context()))

	rr := httptest.NewRecorder()
	h.GetNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// parsePagination defaults: page=1, size=20
	if repo.getByUserPage != 1 {
		t.Errorf("expected default page 1, got %d", repo.getByUserPage)
	}
	if repo.getByUserSize != 20 {
		t.Errorf("expected default size 20, got %d", repo.getByUserSize)
	}
}

func TestGetNotifications_CustomPagination(t *testing.T) {
	h, repo := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/notifications?page=3&size=10", nil)
	req = req.WithContext(authContext(req.Context()))

	rr := httptest.NewRecorder()
	h.GetNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if repo.getByUserPage != 3 {
		t.Errorf("expected page 3, got %d", repo.getByUserPage)
	}
	if repo.getByUserSize != 10 {
		t.Errorf("expected size 10, got %d", repo.getByUserSize)
	}
}

func TestGetUnreadCount_NoAuth(t *testing.T) {
	h, _ := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/unread-count", nil)
	rr := httptest.NewRecorder()
	h.GetUnreadCount(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestGetUnreadCount_WithAuth(t *testing.T) {
	h, _ := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/unread-count", nil)
	req = req.WithContext(authContext(req.Context()))

	rr := httptest.NewRecorder()
	h.GetUnreadCount(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]int
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := resp["count"]; !ok {
		t.Error("expected 'count' field in response")
	}
}

func TestMarkAsRead_NoAuth(t *testing.T) {
	h, _ := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodPut, "/api/notifications/1/read", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("seq", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.MarkAsRead(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestMarkAsRead_InvalidSeq(t *testing.T) {
	h, _ := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodPut, "/api/notifications/abc/read", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("seq", "abc")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = authContext(ctx)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.MarkAsRead(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var apiErr model.APIError
	if err := json.NewDecoder(rr.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if apiErr.Code != "INVALID_SEQ" {
		t.Errorf("expected error code INVALID_SEQ, got %s", apiErr.Code)
	}
}

func TestMarkAllAsRead_NoAuth(t *testing.T) {
	h, _ := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodPut, "/api/notifications/read-all", nil)
	rr := httptest.NewRecorder()
	h.MarkAllAsRead(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestMarkAllAsRead_WithAuth(t *testing.T) {
	h, _ := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodPut, "/api/notifications/read-all", nil)
	req = req.WithContext(authContext(req.Context()))

	rr := httptest.NewRecorder()
	h.MarkAllAsRead(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}
