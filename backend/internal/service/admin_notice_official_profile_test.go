package service

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func boolPtr(v bool) *bool { return &v }

// expectNoticeInsertByline primes a no-attachment Create and asserts the
// stored byline: REG_NAME, USR_SEQ and OFFICIAL_PROFILE_YN.
func expectNoticeInsertByline(mock sqlmock.Sqlmock, regName string, usrSeq int, officialYN string) {
	mock.ExpectQuery(`SELECT MAX\(SEQ\) FROM WEO_BOARDBBS`).
		WillReturnRows(sqlmock.NewRows([]string{"MAX(SEQ)"}).AddRow(500))
	mock.ExpectExec(`(?s)INSERT INTO WEO_BOARDBBS.*OFFICIAL_PROFILE_YN`).
		WithArgs(501, "공지", sqlmock.AnyArg(), "본문", sqlmock.AnyArg(), sqlmock.AnyArg(), "N",
			usrSeq, regName, nil, officialYN).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

// expectNoticeUpdateByline asserts the byline arguments of the UPDATE; empty
// values mean "keep the stored value".
func expectNoticeUpdateByline(mock sqlmock.Sqlmock, officialYN, regName string) {
	mock.ExpectExec(`(?s)UPDATE WEO_BOARDBBS.*OFFICIAL_PROFILE_YN = IFNULL\(NULLIF\(\?, ''\), OFFICIAL_PROFILE_YN\).*REG_NAME = IFNULL\(NULLIF\(\?, ''\), REG_NAME\)`).
		WithArgs("수정", sqlmock.AnyArg(), "본문", sqlmock.AnyArg(), sqlmock.AnyArg(), "N", nil, officialYN, regName, 501).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE WEO_FILES`).WillReturnResult(sqlmock.NewResult(0, 0))
}

func expectNoticeAuthorProfile(mock sqlmock.Sqlmock, officialYN, authorName string) {
	mock.ExpectQuery(`(?s)SELECT b.OFFICIAL_PROFILE_YN, IFNULL\(m.USR_NAME, ''\) AS author_name.*LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = b.USR_SEQ`).
		WithArgs(501).
		WillReturnRows(sqlmock.NewRows([]string{"OFFICIAL_PROFILE_YN", "author_name"}).AddRow(officialYN, authorName))
}

func newOfficialProfileServiceTest(t *testing.T) (*AdminNoticeService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	wrapped := sqlx.NewDb(db, "sqlmock")
	return NewAdminNoticeService(repository.NewAdminNoticeRepository(wrapped), repository.NewFileRepository(wrapped), repository.NewAdminFeedCategoryRepository(wrapped)), mock
}

func TestCreateNoticeDefaultsToTheOfficialProfile(t *testing.T) {
	service, mock := newOfficialProfileServiceTest(t)
	expectNoticeInsertByline(mock, OfficialProfileName, 7, "Y")

	if _, err := service.Create("공지", "본문", "홍길동", 7, "N", nil, nil, nil); err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateNoticeWithOfficialProfileTrueStoresTheOfficialName(t *testing.T) {
	service, mock := newOfficialProfileServiceTest(t)
	expectNoticeInsertByline(mock, "대일외고장학회", 7, "Y")

	if _, err := service.Create("공지", "본문", "홍길동", 7, "N", nil, nil, boolPtr(true)); err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateNoticeWithOfficialProfileFalseKeepsTheAdminName(t *testing.T) {
	service, mock := newOfficialProfileServiceTest(t)
	expectNoticeInsertByline(mock, "홍길동", 7, "N")

	if _, err := service.Create("공지", "본문", "홍길동", 7, "N", nil, nil, boolPtr(false)); err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateNoticeWithoutOfficialProfileKeepsTheByline(t *testing.T) {
	service, mock := newOfficialProfileServiceTest(t)
	expectNoticeUpdateByline(mock, "", "")

	if err := service.Update(501, "수정", "본문", "N", nil, nil, nil, "편집자"); err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateNoticeTurningOfficialProfileOnStoresTheOfficialName(t *testing.T) {
	service, mock := newOfficialProfileServiceTest(t)
	expectNoticeUpdateByline(mock, "Y", OfficialProfileName)

	if err := service.Update(501, "수정", "본문", "N", nil, nil, boolPtr(true), "편집자"); err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateNoticeTurningOfficialProfileOffRestoresTheAuthorName(t *testing.T) {
	service, mock := newOfficialProfileServiceTest(t)
	expectNoticeAuthorProfile(mock, "Y", "홍길동")
	expectNoticeUpdateByline(mock, "N", "홍길동")

	if err := service.Update(501, "수정", "본문", "N", nil, nil, boolPtr(false), "편집자"); err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateNoticeTurningOfficialProfileOffFallsBackToTheEditor(t *testing.T) {
	service, mock := newOfficialProfileServiceTest(t)
	expectNoticeAuthorProfile(mock, "Y", "")
	expectNoticeUpdateByline(mock, "N", "편집자")

	if err := service.Update(501, "수정", "본문", "N", nil, nil, boolPtr(false), "편집자"); err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// A post that was never official keeps its stored REG_NAME (legacy bylines
// are not rewritten to the member name).
func TestUpdateNoticeKeepingOfficialProfileOffKeepsTheStoredName(t *testing.T) {
	service, mock := newOfficialProfileServiceTest(t)
	expectNoticeAuthorProfile(mock, "N", "홍길동")
	expectNoticeUpdateByline(mock, "N", "")

	if err := service.Update(501, "수정", "본문", "N", nil, nil, boolPtr(false), "편집자"); err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
