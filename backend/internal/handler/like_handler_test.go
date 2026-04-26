// like_handler_test.go — Unit tests for LikeHandler HTTP endpoints
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
)

// stubLikeRepo is a minimal LikeQuerier stub for handler tests.
type stubLikeRepo struct{}

func (s *stubLikeRepo) HasUserLiked(bbsSeq, usrSeq int) (bool, error) { return false, nil }
func (s *stubLikeRepo) HasAnyLikeRow(bbsSeq, usrSeq int) (bool, error) { return false, nil }
func (s *stubLikeRepo) InsertLike(bbsSeq, usrSeq int) error            { return nil }
func (s *stubLikeRepo) SetLikeOpenByUser(bbsSeq, usrSeq int, openYN string) error {
	return nil
}

// stubFeedRepo is a minimal FeedQuerier stub for handler tests.
type stubFeedRepo struct{}

func (s *stubFeedRepo) GetNotices(cursor, size, heroSeq, userSeq int) ([]model.NoticeItem, error) {
	return nil, nil
}
func (s *stubFeedRepo) GetHeroNotice() (*model.NoticeItem, error)        { return nil, nil }
func (s *stubFeedRepo) GetNoticeDetail(seq int) (*model.NoticeDetail, error) { return nil, nil }
func (s *stubFeedRepo) IncrementHit(seq int) error                       { return nil }
func (s *stubFeedRepo) GetLikeCount(seq int) (int, error)                { return 5, nil }
func (s *stubFeedRepo) GetCommentCount(seq int) (int, error)             { return 0, nil }
func (s *stubFeedRepo) GetPrevPost(seq int) (*model.PostSibling, error)  { return nil, nil }
func (s *stubFeedRepo) GetNextPost(seq int) (*model.PostSibling, error)  { return nil, nil }
func (s *stubFeedRepo) GetFilesByPost(seq int) ([]model.FileRecord, error) { return nil, nil }
func (s *stubFeedRepo) GetPostOwnerSeq(seq int) (int, error)             { return 0, nil }

func newTestLikeHandler() *LikeHandler {
	svc := service.NewLikeService(&stubLikeRepo{}, &stubFeedRepo{})
	return NewLikeHandler(svc)
}

func TestToggleLike_InvalidSeq(t *testing.T) {
	h := newTestLikeHandler()

	tests := []struct {
		name   string
		seqVal string
	}{
		{"empty seq", ""},
		{"non-numeric seq", "abc"},
		{"zero seq", "0"},
		{"negative seq", "-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/feed/"+tc.seqVal+"/like", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("seq", tc.seqVal)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			h.ToggleLike(rr, req)

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
		})
	}
}

func TestToggleLike_NoAuthContext(t *testing.T) {
	h := newTestLikeHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/feed/42/like", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("seq", "42")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.ToggleLike(rr, req)

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

func TestToggleLike_Success(t *testing.T) {
	h := newTestLikeHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/feed/42/like", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("seq", "42")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = middleware.SetAuthUser(ctx, &model.AuthUser{
		USRSeq:    1,
		USRID:     "testuser",
		USRName:   "Test User",
		USRStatus: "BBB",
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.ToggleLike(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp model.LikeToggleResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Liked {
		t.Error("expected liked to be true")
	}
}
