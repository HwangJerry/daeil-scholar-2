package repository

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func newOfficialProfileRepoMock(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlmock"), mock
}

var officialProfileSelect = regexp.QuoteMeta(officialProfileColumn)

func TestInsertNoticeStoresTheOfficialProfileFlag(t *testing.T) {
	db, mock := newOfficialProfileRepoMock(t)
	repo := NewAdminNoticeRepository(db)
	for _, tc := range []struct{ in, want string }{{"Y", "Y"}, {"N", "N"}, {"", "N"}, {"x", "N"}} {
		mock.ExpectQuery(`SELECT MAX\(SEQ\) FROM WEO_BOARDBBS`).
			WillReturnRows(sqlmock.NewRows([]string{"MAX(SEQ)"}).AddRow(10))
		mock.ExpectExec(`(?s)INSERT INTO WEO_BOARDBBS.*FEED_CATEGORY_SEQ, OFFICIAL_PROFILE_YN\).*NOW\(\), 0, 0, \?, \?\)`).
			WithArgs(11, "제목", "", "", "", "", "N", 3, "이름", nil, tc.want).
			WillReturnResult(sqlmock.NewResult(1, 1))
		if _, err := repo.InsertNotice(&model.AdminNoticeInsert{
			Subject: "제목", IsPinned: "N", RegName: "이름", USRSeq: 3, OfficialProfileYN: tc.in,
		}); err != nil {
			t.Fatalf("InsertNotice(%q) error = %v", tc.in, err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateNoticeKeepsTheBylineWhenEmpty(t *testing.T) {
	db, mock := newOfficialProfileRepoMock(t)
	mock.ExpectExec(`(?s)OFFICIAL_PROFILE_YN = IFNULL\(NULLIF\(\?, ''\), OFFICIAL_PROFILE_YN\),\s+REG_NAME = IFNULL\(NULLIF\(\?, ''\), REG_NAME\)\s+WHERE SEQ = \? AND GATE = 'NOTICE'`).
		WithArgs("제목", "", "", "", "", "N", nil, "", "", 9).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := NewAdminNoticeRepository(db).UpdateNotice(9, &model.AdminNoticeInsert{Subject: "제목", IsPinned: "N"}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetNoticeAuthorProfileReadsFlagAndMemberName(t *testing.T) {
	db, mock := newOfficialProfileRepoMock(t)
	repo := NewAdminNoticeRepository(db)
	query := `(?s)SELECT b.OFFICIAL_PROFILE_YN, IFNULL\(m.USR_NAME, ''\) AS author_name\s+FROM WEO_BOARDBBS b\s+LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = b.USR_SEQ\s+WHERE b.SEQ = \? AND b.GATE = 'NOTICE'`
	mock.ExpectQuery(query).WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"OFFICIAL_PROFILE_YN", "author_name"}).AddRow("Y", "홍길동"))
	mock.ExpectQuery(query).WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"OFFICIAL_PROFILE_YN", "author_name"}))

	profile, err := repo.GetNoticeAuthorProfile(9)
	if err != nil || profile == nil || profile.OfficialProfileYN != "Y" || profile.AuthorName != "홍길동" {
		t.Fatalf("GetNoticeAuthorProfile(9) = %+v, %v", profile, err)
	}
	missing, err := repo.GetNoticeAuthorProfile(10)
	if err != nil || missing != nil {
		t.Fatalf("missing post = %+v, %v", missing, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminNoticeReadsExposeOfficialProfile(t *testing.T) {
	db, mock := newOfficialProfileRepoMock(t)
	repo := NewAdminNoticeRepository(db)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM WEO_BOARDBBS b`).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery(`(?s)SELECT b.SEQ, b.SUBJECT, b.REG_DATE, b.REG_NAME.*` + officialProfileSelect).
		WillReturnRows(sqlmock.NewRows([]string{"SEQ", "REG_NAME", "official_profile"}).AddRow(1, "대일외고장학회", int64(1)))
	mock.ExpectQuery(`(?s)SELECT b.SEQ, b.SUBJECT, b.CONTENTS.*` + officialProfileSelect + `.*WHERE b.SEQ = \?`).WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"SEQ", "official_profile"}).AddRow(1, []byte("0")))

	rows, _, err := repo.GetNotices(1, 20, "", 0)
	if err != nil || len(rows) != 1 || !rows[0].OfficialProfile {
		t.Fatalf("GetNotices = %+v, %v", rows, err)
	}
	detail, err := repo.GetNoticeForEdit(1)
	if err != nil || detail == nil || detail.OfficialProfile {
		t.Fatalf("GetNoticeForEdit = %+v, %v", detail, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPublicFeedReadsExposeOfficialProfile(t *testing.T) {
	db, mock := newOfficialProfileRepoMock(t)
	repo := NewFeedRepository(db)
	mock.ExpectQuery(`(?s)SELECT b.SEQ, b.SUBJECT.*` + officialProfileSelect + `.*WEO_BOARDLIKE.*ORDER BY`).
		WillReturnRows(sqlmock.NewRows([]string{"SEQ", "official_profile"}).AddRow(1, []byte("1")).AddRow(2, []byte("0")))
	mock.ExpectQuery(`(?s)SELECT b.SEQ, b.SUBJECT.*` + officialProfileSelect + `.*LIMIT 1`).
		WillReturnRows(sqlmock.NewRows([]string{"SEQ", "official_profile"}).AddRow(1, int64(1)))
	mock.ExpectQuery(`(?s)SELECT b.SEQ, b.SUBJECT, IFNULL\(b.CONTENTS,''\).*` + officialProfileSelect).WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"SEQ", "official_profile"}).AddRow(1, int64(1)))

	items, err := repo.GetNotices(0, 20, 0, 0)
	if err != nil || len(items) != 2 || !items[0].OfficialProfile || items[1].OfficialProfile {
		t.Fatalf("GetNotices = %+v, %v", items, err)
	}
	hero, err := repo.GetHeroNotice()
	if err != nil || hero == nil || !hero.OfficialProfile {
		t.Fatalf("GetHeroNotice = %+v, %v", hero, err)
	}
	detail, err := repo.GetNoticeDetail(1)
	if err != nil || detail == nil || !detail.OfficialProfile {
		t.Fatalf("GetNoticeDetail = %+v, %v", detail, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
