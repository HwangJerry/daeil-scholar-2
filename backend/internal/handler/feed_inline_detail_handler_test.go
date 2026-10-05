package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/presenter"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
)

func newInlineFeedRouter(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })
	repo := repository.NewFeedRepository(sqlx.NewDb(rawDB, "sqlmock"))
	h := NewFeedHandler(
		service.NewFeedService(repo, cache.New(time.Minute, time.Minute)),
		service.NewLikeService(nil, repo),
		service.NewFeedInlineDetailService(repo),
		presenter.NewFeedPresenter(),
	)
	r := chi.NewRouter()
	r.Get("/api/feed", h.GetFeed)
	r.Post("/api/feed/{seq}/view", h.RecordView)
	return r, mock
}

var inlineNoticeColumns = []string{"SEQ", "SUBJECT", "SUMMARY", "THUMBNAIL_URL", "REG_DATE", "REG_NAME", "HIT", "IS_PINNED",
	"official_profile", "like_cnt", "comment_cnt", "user_liked", "category", "category_name"}

func expectInlineFeedPage(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`(?s)FROM WEO_BOARDBBS b.*ORDER BY`).
		WillReturnRows(sqlmock.NewRows(inlineNoticeColumns).AddRow(3, "제목", "요약", "", "2026-10-01", "관리자", 7, "N", 0, 1, 5, 1, "notice", "공지"))
	mock.ExpectQuery(`FROM ALUMNI_FEED_CATEGORY`).
		WillReturnRows(sqlmock.NewRows([]string{"FC_CODE", "FC_NAME"}).AddRow("notice", "공지"))
}

func TestWantsInlineDetail(t *testing.T) {
	for query, want := range map[string]bool{
		"": false, "include=detail": true, "include=foo,detail": true, "include=foo&include=detail": true,
		"include=details": false, "include=": false,
	} {
		r := httptest.NewRequest(http.MethodGet, "/api/feed?"+query, nil)
		if got := wantsInlineDetail(r); got != want {
			t.Errorf("wantsInlineDetail(%q) = %v, want %v", query, got, want)
		}
	}
}

func TestGetFeedWithoutIncludeRunsNoDetailQueries(t *testing.T) {
	router, mock := newInlineFeedRouter(t)
	expectInlineFeedPage(mock)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/feed", nil))
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "contentHtml") || strings.Contains(rec.Body.String(), "comments") {
		t.Fatalf("plain feed = %d %s", rec.Code, rec.Body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetFeedIncludeDetailAddsBodyCommentsAndFiles(t *testing.T) {
	router, mock := newInlineFeedRouter(t)
	expectInlineFeedPage(mock)
	mock.ExpectQuery(`FROM WEO_BOARDBBS\s+WHERE SEQ IN \(\?\)`).WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"SEQ", "CONTENTS", "CONTENT_FORMAT"}).AddRow(3, "PHA+aGk8L3A+", "MARKDOWN"))
	mock.ExpectQuery(`FROM WEO_BOARDCOMAND\s+WHERE JOIN_SEQ IN \(\?\)`).WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"BC_SEQ", "JOIN_SEQ", "USR_SEQ", "NICKNAME", "CONTENTS", "REG_DATE"}).
			AddRow(2, 3, 9, "동문", "둘", "2026-10-01 09:01").AddRow(1, 3, 9, "동문", "하나", "2026-10-01 09:00"))
	mock.ExpectQuery(`FROM WEO_FILES\s+WHERE F_JOIN_SEQ IN \(\?\)`).WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"F_SEQ", "F_GATE", "F_JOIN_SEQ", "TYPE_NAME", "FILE_NAME", "FILE_SIZE", "FILE_PATH", "FILE_ORG_NAME", "OPEN_YN"}))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/feed?include=detail", nil))
	body := rec.Body.String()
	for _, want := range []string{`"contentHtml":"\u003cp\u003ehi\u003c/p\u003e"`, `"contentFormat":"MARKDOWN"`, `"files":[]`, `"commentsHasMore":false`,
		`"commentCnt":2`, `"likeCnt":1`, `"userLiked":true`, `"hit":7`, `"bcSeq":2`} {
		if !strings.Contains(body, want) {
			t.Fatalf("include=detail body lacks %s: %s", want, body)
		}
	}
	if rec.Code != http.StatusOK || strings.Index(body, `"bcSeq":2`) > strings.Index(body, `"bcSeq":1`) {
		t.Fatalf("include=detail = %d %s", rec.Code, body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordView(t *testing.T) {
	router, mock := newInlineFeedRouter(t)
	hitQuery := `SELECT HIT FROM WEO_BOARDBBS\s+WHERE SEQ = \? AND GATE = 'NOTICE' AND OPEN_YN = 'Y'`
	mock.ExpectQuery(hitQuery).WithArgs(5).WillReturnRows(sqlmock.NewRows([]string{"HIT"}).AddRow(10))
	mock.ExpectExec(`UPDATE WEO_BOARDBBS SET HIT = HIT \+ 1 WHERE SEQ = \?`).WithArgs(5).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(hitQuery).WithArgs(5).WillReturnRows(sqlmock.NewRows([]string{"HIT"}).AddRow(11))
	mock.ExpectQuery(hitQuery).WithArgs(6).WillReturnRows(sqlmock.NewRows([]string{"HIT"}))

	for _, tc := range []struct {
		path string
		code int
		body string
	}{
		{"/api/feed/5/view", http.StatusOK, `{"hit":11}`},
		{"/api/feed/6/view", http.StatusNotFound, `"NOT_FOUND"`},
		{"/api/feed/0/view", http.StatusBadRequest, `"INVALID_SEQ"`},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tc.path, nil))
		if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.body) {
			t.Fatalf("%s = %d %s", tc.path, rec.Code, rec.Body)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
