package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/golden"
)

// TestGoldenFeedInlineDetail drives include=detail and the explicit view
// counter through the real router on MariaDB 10.1: each feed item and the hero
// carry exactly the detail endpoint's body/files and the comments endpoint's
// list, likes match the caller, no view is counted until POST /view, and the
// plain responses (and the cached hero) stay untouched.
func TestGoldenFeedInlineDetail(t *testing.T) {
	s := newGoldenServer(t)
	operator := defaultGoldenMember
	seedGoldenMemberAs(t, s.db, operator)
	goldenExec(t, s.db, `INSERT INTO ALUMNI_ADMIN_ROLE (USR_SEQ, ADMIN_ROLE, CREATED_AT, UPDATED_AT)
		VALUES (?, 'operator', NOW(), NOW())`, operator.seq)
	_, session := s.loginAs(t, operator, "android")
	token := session.AccessToken

	plainSeq := s.createGoldenNotice(t, token, map[string]any{"subject": "합성 빈 공지", "contentMd": "본문 B"})
	richSeq := s.createGoldenNotice(t, token, map[string]any{"subject": "합성 펼침 공지", "contentMd": "**굵게** 본문\n\n- 하나\n- 둘"})
	goldenExec(t, s.db, `INSERT INTO WEO_FILES (F_GATE, F_JOIN_SEQ, TYPE_NAME, FILE_NAME, FILE_SIZE, FILE_PATH, FILE_ORG_NAME, OPEN_YN, REG_DATE)
		VALUES ('BB', ?, 'application/pdf', 'synthetic.pdf', '1024', '/files/notice', '안내문.pdf', 'Y', NOW())`, richSeq)
	commentPath := fmt.Sprintf("/api/feed/%d/comments", richSeq)
	var commentSeqs []int
	for _, text := range []string{"첫 댓글", "지울 댓글", "셋째 댓글"} {
		body := s.request(t, http.MethodPost, commentPath, map[string]string{"contents": text}, token, "android", "100", http.StatusCreated)
		commentSeqs = append(commentSeqs, decodeGolden[model.Comment](t, body).BCSeq)
	}
	s.request(t, http.MethodDelete, fmt.Sprintf("%s/%d", commentPath, commentSeqs[1]), nil, token, "android", "100", http.StatusNoContent)
	s.request(t, http.MethodPost, fmt.Sprintf("/api/feed/%d/like", richSeq), nil, token, "android", "100", http.StatusOK)

	plainHero := s.request(t, http.MethodGet, "/api/feed/hero", nil, token, "ios", "100", http.StatusOK)

	t.Run("feed", func(t *testing.T) {
		plain := s.request(t, http.MethodGet, "/api/feed", nil, token, "ios", "100", http.StatusOK)
		assertNoInlineDetail(t, plain)

		body := s.request(t, http.MethodGet, "/api/feed?include=detail", nil, token, "ios", "100", http.StatusOK)
		golden.Assert(t, "feed_inline_detail", body, "regDate")
		items := inlineItems(t, body)
		if len(items) != 2 || items[0].SEQ != richSeq || items[1].SEQ != plainSeq {
			t.Fatalf("include=detail must keep the page: %s", body)
		}
		rich, empty := items[0], items[1]
		if rich.LikeCnt != 1 || !rich.UserLiked || rich.CommentCnt != 2 || len(rich.Comments) != 2 ||
			rich.CommentsHasMore || len(rich.Files) != 1 || rich.ContentFormat != "MARKDOWN" {
			t.Fatalf("rich item: %s", body)
		}
		if empty.Files == nil || empty.Comments == nil || len(empty.Files) != 0 || len(empty.Comments) != 0 || empty.CommentCnt != 0 {
			t.Fatalf("an item without files or comments must carry empty arrays: %s", body)
		}

		anonymous := inlineItems(t, s.request(t, http.MethodGet, "/api/feed?include=detail", nil, "", "ios", "100", http.StatusOK))
		if anonymous[0].UserLiked || anonymous[0].LikeCnt != 1 || len(anonymous[0].Comments) != 2 {
			t.Fatalf("anonymous include=detail = %+v", anonymous[0])
		}
	})
	t.Run("hero", func(t *testing.T) {
		body := s.request(t, http.MethodGet, "/api/feed/hero?include=detail", nil, token, "ios", "100", http.StatusOK)
		golden.Assert(t, "feed_hero_inline_detail", body, "regDate")
		hero := decodeInlineItem(t, body)
		if hero.SEQ != richSeq || !hero.UserLiked || hero.LikeCnt != 1 || hero.CommentCnt != 2 || len(hero.Comments) != 2 || len(hero.Files) != 1 {
			t.Fatalf("hero include=detail: %s", body)
		}
		anonymous := decodeInlineItem(t, s.request(t, http.MethodGet, "/api/feed/hero?include=detail", nil, "", "ios", "100", http.StatusOK))
		if anonymous.UserLiked || anonymous.LikeCnt != 1 {
			t.Fatalf("anonymous hero include=detail = %+v", anonymous)
		}
		again := s.request(t, http.MethodGet, "/api/feed/hero", nil, token, "ios", "100", http.StatusOK)
		if !bytes.Equal(again, plainHero) {
			t.Fatalf("include=detail must not change the cached plain hero:\n%s\n%s", plainHero, again)
		}
		assertNoInlineDetail(t, again)
	})
	t.Run("no_view_counted", func(t *testing.T) {
		goldenCount(t, s.db, 0, `SELECT HIT FROM WEO_BOARDBBS WHERE SEQ = ?`, richSeq)
		goldenCount(t, s.db, 0, `SELECT HIT FROM WEO_BOARDBBS WHERE SEQ = ?`, plainSeq)
	})
	t.Run("view", func(t *testing.T) {
		viewPath := fmt.Sprintf("/api/feed/%d/view", richSeq)
		for want := 1; want <= 2; want++ {
			body := s.request(t, http.MethodPost, viewPath, nil, "", "ios", "100", http.StatusOK)
			if got := decodeGolden[model.NoticeViewResponse](t, body); got.Hit != want {
				t.Fatalf("view %d: %s", want, body)
			}
		}
		golden.Assert(t, "feed_view", s.request(t, http.MethodPost, viewPath, nil, token, "android", "100", http.StatusOK))
		s.request(t, http.MethodPost, "/api/feed/999999/view", nil, "", "ios", "100", http.StatusNotFound)
	})
	t.Run("matches_detail_and_comments", func(t *testing.T) {
		items := inlineItems(t, s.request(t, http.MethodGet, "/api/feed?include=detail", nil, token, "ios", "100", http.StatusOK))
		for _, item := range items {
			detail := decodeGolden[model.NoticeDetail](t, s.request(t, http.MethodGet, fmt.Sprintf("/api/feed/%d", item.SEQ), nil, token, "ios", "100", http.StatusOK))
			comments := decodeGolden[[]model.Comment](t, s.request(t, http.MethodGet, fmt.Sprintf("/api/feed/%d/comments", item.SEQ), nil, token, "ios", "100", http.StatusOK))
			if item.ContentHtml != detail.ContentHtml || item.ContentFormat != detail.ContentFormat ||
				item.LikeCnt != detail.LikeCnt || item.CommentCnt != detail.CommentCnt || item.UserLiked != detail.UserLiked {
				t.Fatalf("post %d inline = %+v, detail = %+v", item.SEQ, item.NoticeInlineDetail, detail)
			}
			if !reflect.DeepEqual(item.Comments, comments) || (len(detail.Files) > 0 && !reflect.DeepEqual(item.Files, detail.Files)) {
				t.Fatalf("post %d comments/files differ: %+v vs %+v", item.SEQ, item.Comments, comments)
			}
		}
		goldenCount(t, s.db, 4, `SELECT HIT FROM WEO_BOARDBBS WHERE SEQ = ?`, richSeq)
	})
	t.Run("view_unpublished", func(t *testing.T) {
		s.request(t, http.MethodDelete, fmt.Sprintf("/api/admin/feed/%d", plainSeq), nil, token, "android", "100", http.StatusNoContent)
		s.request(t, http.MethodPost, fmt.Sprintf("/api/feed/%d/view", plainSeq), nil, "", "ios", "100", http.StatusNotFound)
	})
}

func inlineItems(t *testing.T, body []byte) []model.NoticeItem {
	t.Helper()
	feed := decodeGolden[model.FeedResponse](t, body)
	items := make([]model.NoticeItem, 0, len(feed.Items))
	for _, item := range feed.Items {
		if item.NoticeItem == nil || item.NoticeInlineDetail == nil {
			t.Fatalf("every notice item must carry inline detail: %s", body)
		}
		items = append(items, *item.NoticeItem)
	}
	return items
}

func decodeInlineItem(t *testing.T, body []byte) model.NoticeItem {
	t.Helper()
	item := decodeGolden[model.NoticeItem](t, body)
	if item.NoticeInlineDetail == nil {
		t.Fatalf("hero must carry inline detail: %s", body)
	}
	return item
}

func assertNoInlineDetail(t *testing.T, body []byte) {
	t.Helper()
	for _, key := range []string{`"contentHtml"`, `"comments"`, `"files"`, `"commentsHasMore"`, `"contentFormat"`} {
		if bytes.Contains(body, []byte(key)) {
			t.Fatalf("a response without include=detail must not carry %s: %s", key, body)
		}
	}
	var probe any
	if err := json.Unmarshal(body, &probe); err != nil {
		t.Fatal(err)
	}
}
