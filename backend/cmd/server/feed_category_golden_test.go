package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/golden"
)

// TestGoldenFeedCategories drives feed categories through the real router: an
// operator adds a category and files notices under it, the public feed shows
// each item's category (NULL resolves to the default) plus the open tabs in
// order, and the admin API refuses the rule-breaking changes with the Korean
// messages the admin UI shows.
func TestGoldenFeedCategories(t *testing.T) {
	s := newGoldenServer(t)
	operator := defaultGoldenMember
	seedGoldenMemberAs(t, s.db, operator)
	goldenExec(t, s.db, `INSERT INTO ALUMNI_ADMIN_ROLE (USR_SEQ, ADMIN_ROLE, CREATED_AT, UPDATED_AT)
		VALUES (?, 'operator', NOW(), NOW())`, operator.seq)
	_, session := s.loginAs(t, operator, "android")
	token := session.AccessToken

	body := s.request(t, http.MethodPost, "/api/admin/feed-categories", map[string]any{"name": " 장학 ", "openYn": "Y"}, token, "android", "100", http.StatusCreated)
	created := decodeGolden[model.AdminFeedCategory](t, body)
	if created.Seq != 3 || created.Code != "c3" || created.Name != "장학" || created.SortOrder != 3 {
		t.Fatalf("created category: %s", body)
	}
	golden.Assert(t, "admin_feed_category_created", body)

	s.request(t, http.MethodPut, "/api/admin/feed-categories/order", map[string]any{"seqs": []int{3, 1, 2}}, token, "android", "100", http.StatusNoContent)
	s.request(t, http.MethodPut, "/api/admin/feed-categories/2", map[string]any{"name": "기타", "openYn": "N"}, token, "android", "100", http.StatusOK)

	legacySeq := s.createGoldenNotice(t, token, map[string]any{"subject": "합성 기본 공지", "contentMd": "본문"})
	scholarshipSeq := s.createGoldenNotice(t, token, map[string]any{"subject": "합성 장학 공지", "contentMd": "본문", "categorySeq": 3})
	hiddenSeq := s.createGoldenNotice(t, token, map[string]any{"subject": "합성 기타 공지", "contentMd": "본문", "categorySeq": 2})
	body = s.request(t, http.MethodPost, "/api/admin/feed", map[string]any{"subject": "없는 카테고리", "contentMd": "본문", "categorySeq": 99}, token, "android", "100", http.StatusBadRequest)
	assertGoldenError(t, body, "UNKNOWN_CATEGORY")

	t.Run("feed", func(t *testing.T) {
		body := s.request(t, http.MethodGet, "/api/feed", nil, "", "ios", "100", http.StatusOK)
		feed := decodeGolden[model.FeedResponse](t, body)
		if len(feed.Items) != 3 || len(feed.Categories) != 2 || feed.Categories[0].Code != "c3" || feed.Categories[1].Code != "notice" {
			t.Fatalf("feed must list items and only open categories in order: %s", body)
		}
		golden.Assert(t, "feed_with_categories", body)
	})
	t.Run("feed_detail", func(t *testing.T) {
		body := s.request(t, http.MethodGet, fmt.Sprintf("/api/feed/%d", legacySeq), nil, "", "ios", "100", http.StatusOK)
		detail := decodeGolden[model.NoticeDetail](t, body)
		if detail.Category != "notice" || detail.CategoryName != "공지" || detail.CategorySeq != 0 {
			t.Fatalf("a post without a category must read as the default: %s", body)
		}
		golden.Assert(t, "feed_detail_default_category", body)
	})
	t.Run("feed_hero", func(t *testing.T) {
		body := s.request(t, http.MethodGet, "/api/feed/hero", nil, "", "ios", "100", http.StatusOK)
		hero := decodeGolden[model.NoticeItem](t, body)
		if hero.SEQ != hiddenSeq || hero.Category != "etc" || hero.CategoryName != "기타" {
			t.Fatalf("a hidden category's post keeps its category name: %s", body)
		}
	})
	t.Run("admin_list", func(t *testing.T) {
		body := s.request(t, http.MethodGet, "/api/admin/feed-categories", nil, token, "android", "100", http.StatusOK)
		golden.Assert(t, "admin_feed_categories", body)
		body = s.request(t, http.MethodGet, "/api/admin/feed?category=3", nil, token, "android", "100", http.StatusOK)
		list := decodeGolden[struct {
			Items []model.AdminNoticeRow `json:"items"`
			Total int                    `json:"total"`
		}](t, body)
		if list.Total != 1 || list.Items[0].SEQ != scholarshipSeq || list.Items[0].CategoryName != "장학" {
			t.Fatalf("category filter: %s", body)
		}
		golden.Assert(t, "admin_feed_list_by_category", body)
	})
	t.Run("rules", func(t *testing.T) {
		body := s.request(t, http.MethodPost, "/api/admin/feed-categories", map[string]any{"name": "장학"}, token, "android", "100", http.StatusBadRequest)
		golden.Assert(t, "admin_feed_category_duplicate_400", body)
		body = s.request(t, http.MethodPut, "/api/admin/feed-categories/1", map[string]any{"name": "공지", "openYn": "N"}, token, "android", "100", http.StatusBadRequest)
		assertGoldenError(t, body, "DEFAULT_CATEGORY_HIDE")
		body = s.request(t, http.MethodDelete, "/api/admin/feed-categories/1", nil, token, "android", "100", http.StatusBadRequest)
		assertGoldenError(t, body, "DEFAULT_CATEGORY_DELETE")
		body = s.request(t, http.MethodDelete, "/api/admin/feed-categories/3", nil, token, "android", "100", http.StatusConflict)
		golden.Assert(t, "admin_feed_category_has_posts_409", body)
	})
	t.Run("delete_with_move", func(t *testing.T) {
		s.request(t, http.MethodDelete, "/api/admin/feed-categories/3", map[string]any{"moveToSeq": 1}, token, "android", "100", http.StatusNoContent)
		goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_FEED_CATEGORY WHERE FC_SEQ = 3`)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_BOARDBBS WHERE SEQ = ? AND FEED_CATEGORY_SEQ = 1`, scholarshipSeq)
		// A legacy post is reclassified without its content being rewritten.
		s.request(t, http.MethodPut, fmt.Sprintf("/api/admin/feed/%d/category", legacySeq), map[string]any{"categorySeq": 2}, token, "android", "100", http.StatusNoContent)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_BOARDBBS WHERE SEQ = ? AND FEED_CATEGORY_SEQ = 2`, legacySeq)
	})
}

func (s *goldenServer) createGoldenNotice(t *testing.T, token string, body map[string]any) int {
	t.Helper()
	created := s.request(t, http.MethodPost, "/api/admin/feed", body, token, "android", "100", http.StatusCreated)
	seq := decodeGolden[map[string]int](t, created)["seq"]
	if seq <= 0 {
		t.Fatalf("notice create must return its seq: %s", created)
	}
	return seq
}
