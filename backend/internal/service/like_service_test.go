// like_service_test.go — Unit tests for LikeService toggle state machine
package service

import (
	"errors"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

// mockLikeRepo implements repository.LikeQuerier for testing.
type mockLikeRepo struct {
	hasUserLiked  bool
	hasAnyLikeRow bool
	insertErr     error
	setOpenErr    error
	hasLikedErr   error
	hasAnyErr     error
	insertCalled  bool
	setOpenCalls  []string // records openYN values passed to SetLikeOpenByUser
}

func (m *mockLikeRepo) HasUserLiked(bbsSeq int, usrSeq int) (bool, error) {
	return m.hasUserLiked, m.hasLikedErr
}

func (m *mockLikeRepo) HasAnyLikeRow(bbsSeq int, usrSeq int) (bool, error) {
	return m.hasAnyLikeRow, m.hasAnyErr
}

func (m *mockLikeRepo) InsertLike(bbsSeq int, usrSeq int) error {
	m.insertCalled = true
	return m.insertErr
}

func (m *mockLikeRepo) SetLikeOpenByUser(bbsSeq int, usrSeq int, openYN string) error {
	m.setOpenCalls = append(m.setOpenCalls, openYN)
	return m.setOpenErr
}

// mockFeedRepoForLike implements the FeedQuerier methods needed by LikeService.
type mockFeedRepoForLike struct {
	likeCount    int
	likeCountErr error
}

func (m *mockFeedRepoForLike) GetNotices(cursor int, size int, heroSeq int, userSeq int) ([]model.NoticeItem, error) {
	return nil, nil
}
func (m *mockFeedRepoForLike) GetHeroNotice() (*model.NoticeItem, error) { return nil, nil }
func (m *mockFeedRepoForLike) GetNoticeDetail(seq int) (*model.NoticeDetail, error) {
	return nil, nil
}
func (m *mockFeedRepoForLike) IncrementHit(seq int) error { return nil }
func (m *mockFeedRepoForLike) GetLikeCount(seq int) (int, error) {
	return m.likeCount, m.likeCountErr
}
func (m *mockFeedRepoForLike) GetCommentCount(seq int) (int, error) { return 0, nil }
func (m *mockFeedRepoForLike) GetPrevPost(seq int) (*model.PostSibling, error) {
	return nil, nil
}
func (m *mockFeedRepoForLike) GetNextPost(seq int) (*model.PostSibling, error) {
	return nil, nil
}
func (m *mockFeedRepoForLike) GetFilesByPost(seq int) ([]model.FileRecord, error) {
	return nil, nil
}
func (m *mockFeedRepoForLike) GetPostOwnerSeq(seq int) (int, error) { return 0, nil }

func TestToggleLike_FirstLike(t *testing.T) {
	likeRepo := &mockLikeRepo{hasUserLiked: false, hasAnyLikeRow: false}
	feedRepo := &mockFeedRepoForLike{likeCount: 1}
	svc := &LikeService{likeRepo: likeRepo, feedRepo: feedRepo, notifier: nil}

	resp, err := svc.ToggleLike(100, 1, "TestUser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Liked {
		t.Error("expected Liked=true for first like")
	}
	if !likeRepo.insertCalled {
		t.Error("expected InsertLike to be called")
	}
	if len(likeRepo.setOpenCalls) != 0 {
		t.Error("expected SetLikeOpenByUser not to be called")
	}
	if resp.LikeCnt != 1 {
		t.Errorf("expected LikeCnt=1, got %d", resp.LikeCnt)
	}
}

func TestToggleLike_Unlike(t *testing.T) {
	likeRepo := &mockLikeRepo{hasUserLiked: true}
	feedRepo := &mockFeedRepoForLike{likeCount: 0}
	svc := &LikeService{likeRepo: likeRepo, feedRepo: feedRepo, notifier: nil}

	resp, err := svc.ToggleLike(100, 1, "TestUser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Liked {
		t.Error("expected Liked=false for unlike")
	}
	if len(likeRepo.setOpenCalls) != 1 || likeRepo.setOpenCalls[0] != "N" {
		t.Errorf("expected SetLikeOpenByUser called with 'N', got %v", likeRepo.setOpenCalls)
	}
	if likeRepo.insertCalled {
		t.Error("expected InsertLike not to be called")
	}
}

func TestToggleLike_ReLike(t *testing.T) {
	likeRepo := &mockLikeRepo{hasUserLiked: false, hasAnyLikeRow: true}
	feedRepo := &mockFeedRepoForLike{likeCount: 2}
	svc := &LikeService{likeRepo: likeRepo, feedRepo: feedRepo, notifier: nil}

	resp, err := svc.ToggleLike(100, 1, "TestUser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Liked {
		t.Error("expected Liked=true for re-like")
	}
	if len(likeRepo.setOpenCalls) != 1 || likeRepo.setOpenCalls[0] != "Y" {
		t.Errorf("expected SetLikeOpenByUser called with 'Y', got %v", likeRepo.setOpenCalls)
	}
	if likeRepo.insertCalled {
		t.Error("expected InsertLike not to be called for re-like")
	}
}

func TestToggleLike_NotifierNilSafe(t *testing.T) {
	likeRepo := &mockLikeRepo{hasUserLiked: false, hasAnyLikeRow: false}
	feedRepo := &mockFeedRepoForLike{likeCount: 1}
	svc := &LikeService{likeRepo: likeRepo, feedRepo: feedRepo, notifier: nil}

	resp, err := svc.ToggleLike(100, 1, "TestUser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Liked {
		t.Error("expected Liked=true")
	}
	// Test passes if no panic occurs with nil notifier
}

func TestToggleLike_HasUserLikedError(t *testing.T) {
	likeRepo := &mockLikeRepo{hasLikedErr: errors.New("db error")}
	feedRepo := &mockFeedRepoForLike{}
	svc := &LikeService{likeRepo: likeRepo, feedRepo: feedRepo, notifier: nil}

	_, err := svc.ToggleLike(100, 1, "TestUser")
	if err == nil {
		t.Fatal("expected error from HasUserLiked failure")
	}
}
