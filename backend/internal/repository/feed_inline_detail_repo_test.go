package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// The include=detail reads must be one IN (...) query per page, never one per
// post, and must keep the detail/comments endpoints' filters and order.
func TestFeedInlineDetailReadsBatchOneQueryPerPage(t *testing.T) {
	db, mock := newOfficialProfileRepoMock(t)
	repo := NewFeedRepository(db)
	seqs := []int{12, 11, 10}

	mock.ExpectQuery(`(?s)SELECT SEQ, IFNULL\(CONTENTS,''\) AS CONTENTS,\s+IFNULL\(CONTENT_FORMAT,'LEGACY'\) AS CONTENT_FORMAT\s+FROM WEO_BOARDBBS\s+WHERE SEQ IN \(\?, \?, \?\)`).
		WithArgs(12, 11, 10).
		WillReturnRows(sqlmock.NewRows([]string{"SEQ", "CONTENTS", "CONTENT_FORMAT"}).AddRow(12, "PHA+", "MARKDOWN"))
	mock.ExpectQuery(`(?s)FROM WEO_BOARDCOMAND\s+WHERE JOIN_SEQ IN \(\?, \?, \?\) AND BC_TYPE = 'B' AND OPEN_YN = 'Y'\s+ORDER BY JOIN_SEQ, SEQ DESC`).
		WithArgs(12, 11, 10).
		WillReturnRows(sqlmock.NewRows([]string{"BC_SEQ", "JOIN_SEQ", "USR_SEQ", "NICKNAME", "CONTENTS", "REG_DATE"}).
			AddRow(5, 12, 7, "동문", "댓글", "2026-10-01 09:00"))
	mock.ExpectQuery(`(?s)FROM WEO_FILES\s+WHERE F_JOIN_SEQ IN \(\?, \?, \?\) AND F_GATE = 'BB' AND OPEN_YN = 'Y'\s+ORDER BY F_JOIN_SEQ, F_SEQ`).
		WithArgs(12, 11, 10).
		WillReturnRows(sqlmock.NewRows([]string{"F_SEQ", "F_GATE", "F_JOIN_SEQ", "TYPE_NAME", "FILE_NAME", "FILE_SIZE", "FILE_PATH", "FILE_ORG_NAME", "OPEN_YN"}))
	mock.ExpectQuery(`(?s)SELECT BBS_SEQ, COUNT\(\*\) AS like_cnt, MAX\(USR_SEQ = \?\) AS user_liked\s+FROM WEO_BOARDLIKE\s+WHERE BBS_SEQ IN \(\?, \?, \?\) AND OPEN_YN = 'Y'\s+GROUP BY BBS_SEQ`).
		WithArgs(7, 12, 11, 10).
		WillReturnRows(sqlmock.NewRows([]string{"BBS_SEQ", "like_cnt", "user_liked"}).AddRow(12, 3, 1))

	bodies, err := repo.GetNoticeBodies(seqs)
	if err != nil || len(bodies) != 1 || bodies[0].ContentFormat != "MARKDOWN" {
		t.Fatalf("bodies = %+v, %v", bodies, err)
	}
	comments, err := repo.GetCommentsByPosts(seqs)
	if err != nil || len(comments) != 1 || comments[0].JoinSeq != 12 || comments[0].RegName != "동문" {
		t.Fatalf("comments = %+v, %v", comments, err)
	}
	files, err := repo.GetFilesByPosts(seqs)
	if err != nil || files == nil || len(files) != 0 {
		t.Fatalf("files = %+v, %v", files, err)
	}
	stats, err := repo.GetLikeStats(seqs, 7)
	if err != nil || len(stats) != 1 || stats[0].LikeCnt != 3 || !stats[0].UserLiked {
		t.Fatalf("stats = %+v, %v", stats, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFeedInlineDetailReadsSkipTheDatabaseForAnEmptyPage(t *testing.T) {
	db, mock := newOfficialProfileRepoMock(t)
	repo := NewFeedRepository(db)
	if bodies, err := repo.GetNoticeBodies(nil); err != nil || bodies == nil {
		t.Fatalf("bodies = %v, %v", bodies, err)
	}
	if comments, err := repo.GetCommentsByPosts(nil); err != nil || comments == nil {
		t.Fatalf("comments = %v, %v", comments, err)
	}
	if files, err := repo.GetFilesByPosts(nil); err != nil || files == nil {
		t.Fatalf("files = %v, %v", files, err)
	}
	if stats, err := repo.GetLikeStats(nil, 1); err != nil || stats == nil {
		t.Fatalf("stats = %v, %v", stats, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetPublishedNoticeHitUsesTheDetailVisibilityFilter(t *testing.T) {
	db, mock := newOfficialProfileRepoMock(t)
	repo := NewFeedRepository(db)
	query := `(?s)SELECT HIT FROM WEO_BOARDBBS\s+WHERE SEQ = \? AND GATE = 'NOTICE' AND OPEN_YN = 'Y'`
	mock.ExpectQuery(query).WithArgs(9).WillReturnRows(sqlmock.NewRows([]string{"HIT"}).AddRow(41))
	mock.ExpectQuery(query).WithArgs(10).WillReturnRows(sqlmock.NewRows([]string{"HIT"}))

	if hit, found, err := repo.GetPublishedNoticeHit(9); err != nil || !found || hit != 41 {
		t.Fatalf("published = %d, %v, %v", hit, found, err)
	}
	if hit, found, err := repo.GetPublishedNoticeHit(10); err != nil || found || hit != 0 {
		t.Fatalf("missing = %d, %v, %v", hit, found, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// NoticeItem embeds the inline detail pointer with db:"-", so the feed query
// still scans into it and leaves the detail nil (plain responses unchanged).
func TestGetNoticesLeavesInlineDetailUnset(t *testing.T) {
	db, mock := newOfficialProfileRepoMock(t)
	mock.ExpectQuery(`(?s)FROM WEO_BOARDBBS b.*WHERE b.GATE = 'NOTICE' AND b.OPEN_YN = 'Y'`).
		WithArgs(0, 11).
		WillReturnRows(sqlmock.NewRows([]string{"SEQ", "SUBJECT", "SUMMARY", "THUMBNAIL_URL", "REG_DATE", "REG_NAME", "HIT", "IS_PINNED",
			"official_profile", "like_cnt", "comment_cnt", "user_liked", "category", "category_name"}).
			AddRow(3, "제목", "", "", "2026-10-01", "관리자", 0, "N", 1, 2, 1, 0, "notice", "공지"))
	notices, err := NewFeedRepository(db).GetNotices(0, 10, 0, 0)
	if err != nil || len(notices) != 1 || notices[0].NoticeInlineDetail != nil || notices[0].LikeCnt != 2 {
		t.Fatalf("notices = %+v, %v", notices, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
