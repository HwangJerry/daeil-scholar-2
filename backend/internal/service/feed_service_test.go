// feed_service_test.go — Unit tests for FeedService GetFeed clamping, cursor, and pagination
package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
)

// mockFeedRepo implements repository.FeedQuerier for testing.
type mockFeedRepo struct {
	notices      []model.NoticeItem
	noticesErr   error
	heroNotice   *model.NoticeItem
	heroErr      error
	detail       *model.NoticeDetail
	detailErr    error
	likeCount    int
	likeCountErr error
	commentCount int
	commentErr   error
	prevPost     *model.PostSibling
	nextPost     *model.PostSibling
	files        []model.FileRecord
	filesErr     error
	ownerSeq     int
}

func (m *mockFeedRepo) GetNotices(cursor int, size int, heroSeq int, userSeq int) ([]model.NoticeItem, error) {
	return m.notices, m.noticesErr
}
func (m *mockFeedRepo) GetHeroNotice() (*model.NoticeItem, error) { return m.heroNotice, m.heroErr }
func (m *mockFeedRepo) GetNoticeDetail(seq int) (*model.NoticeDetail, error) {
	return m.detail, m.detailErr
}
func (m *mockFeedRepo) IncrementHit(seq int) error                         { return nil }
func (m *mockFeedRepo) GetLikeCount(seq int) (int, error)                  { return m.likeCount, m.likeCountErr }
func (m *mockFeedRepo) GetCommentCount(seq int) (int, error)               { return m.commentCount, m.commentErr }
func (m *mockFeedRepo) GetPrevPost(seq int) (*model.PostSibling, error)    { return m.prevPost, nil }
func (m *mockFeedRepo) GetNextPost(seq int) (*model.PostSibling, error)    { return m.nextPost, nil }
func (m *mockFeedRepo) GetFilesByPost(seq int) ([]model.FileRecord, error) { return m.files, m.filesErr }
func (m *mockFeedRepo) GetPostOwnerSeq(seq int) (int, error)               { return m.ownerSeq, nil }

func newTestFeedService(repo *mockFeedRepo) *FeedService {
	cacheStore := cache.New(5*time.Minute, 10*time.Minute)
	return NewFeedService(repo, cacheStore)
}

func TestGetFeed_SizeClampZero(t *testing.T) {
	repo := &mockFeedRepo{notices: []model.NoticeItem{}}
	svc := newTestFeedService(repo)

	resp, err := svc.GetFeed(0, 0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.HasMore {
		t.Error("expected HasMore=false for empty feed")
	}
}

func TestGetFeed_SizeClampLarge(t *testing.T) {
	// Create 21 notices to test clamping at 20
	notices := make([]model.NoticeItem, 21)
	for i := range notices {
		notices[i] = model.NoticeItem{SEQ: 100 - i, Subject: "Test"}
	}
	repo := &mockFeedRepo{notices: notices}
	svc := newTestFeedService(repo)

	resp, err := svc.GetFeed(0, 25, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.HasMore {
		t.Error("expected HasMore=true when notices > clamped size")
	}
	if len(resp.Items) != 20 {
		t.Errorf("expected 20 notice items after clamping, got %d", len(resp.Items))
	}
	for i, item := range resp.Items {
		if item.Type != "notice" || item.NoticeItem == nil {
			t.Errorf("item %d is not a notice-only feed item: %#v", i, item)
		}
	}
}

func TestGetFeed_HasMoreDetermination(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		size     int
		wantMore bool
	}{
		{"exact size", 5, 5, false},
		{"more than size", 6, 5, true},
		{"less than size", 3, 5, false},
		{"empty", 0, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notices := make([]model.NoticeItem, tt.count)
			for i := range notices {
				notices[i] = model.NoticeItem{SEQ: 100 - i}
			}
			repo := &mockFeedRepo{notices: notices}
			svc := newTestFeedService(repo)

			resp, err := svc.GetFeed(0, tt.size, 0, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.HasMore != tt.wantMore {
				t.Errorf("HasMore: got %v, want %v", resp.HasMore, tt.wantMore)
			}
		})
	}
}

func TestGetFeed_NextCursor(t *testing.T) {
	notices := []model.NoticeItem{
		{SEQ: 50, Subject: "A"},
		{SEQ: 45, Subject: "B"},
		{SEQ: 40, Subject: "C"},
	}
	repo := &mockFeedRepo{notices: notices}
	svc := newTestFeedService(repo)

	resp, err := svc.GetFeed(0, 10, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedCursor := "seq_40"
	if resp.NextCursor != expectedCursor {
		t.Errorf("NextCursor: got %q, want %q", resp.NextCursor, expectedCursor)
	}
}

func TestGetFeed_EmptyFeed(t *testing.T) {
	repo := &mockFeedRepo{notices: []model.NoticeItem{}}
	svc := newTestFeedService(repo)

	resp, err := svc.GetFeed(0, 10, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.HasMore {
		t.Error("expected HasMore=false for empty feed")
	}
	if resp.NextCursor != "" {
		t.Errorf("expected empty NextCursor, got %q", resp.NextCursor)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
}

func TestGetFeed_NoticesError(t *testing.T) {
	repo := &mockFeedRepo{noticesErr: errDB}
	svc := newTestFeedService(repo)

	_, err := svc.GetFeed(0, 10, 0, 0)
	if err == nil {
		t.Fatal("expected error from GetNotices failure")
	}
}

func TestGetFeed_SerializesNoticeType(t *testing.T) {
	repo := &mockFeedRepo{notices: []model.NoticeItem{{SEQ: 42, Subject: "Notice"}}}
	svc := newTestFeedService(repo)

	resp, err := svc.GetFeed(0, 10, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal feed response: %v", err)
	}
	if !strings.Contains(string(payload), `"type":"notice"`) {
		t.Fatalf("expected serialized notice type, got %s", payload)
	}
}
