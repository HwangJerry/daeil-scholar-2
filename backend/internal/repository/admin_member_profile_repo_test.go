package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func newAdminMemberRepoMock(t *testing.T) (*AdminMemberRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock")), mock
}

func expectPhoneClaimsApplied(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`SELECT state FROM _migration_journal`).
		WillReturnRows(sqlmock.NewRows([]string{"state"}).AddRow("APPLIED"))
}

func TestAdminUpdateMemberProfileMovesPhoneClaimWithPhone(t *testing.T) {
	repo, mock := newAdminMemberRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_PHONE FROM WEO_MEMBER WHERE USR_SEQ = \? FOR UPDATE`).WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_PHONE"}).AddRow("01011112222"))
	expectPhoneClaimsApplied(mock)
	mock.ExpectExec(`UPDATE AUTH_PHONE_CLAIM`).WithArgs("01033334444", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE WEO_MEMBER\s+LEFT JOIN ALUMNI_VERIFICATION v`).
		WithArgs("홍길동", "01033334444", "m@example.com", "30", "영어", "30", "영어", 42).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	err := repo.UpdateMemberProfile(42, model.AdminMemberProfileUpdate{
		USRName: "홍길동", USRPhone: "01033334444", USREmail: "m@example.com", USRFN: "30", USRDept: "영어",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminUpdateMemberProfileWithoutPhoneKeepsStoredPhoneAndClaim(t *testing.T) {
	repo, mock := newAdminMemberRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_PHONE FROM WEO_MEMBER`).WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_PHONE"}).AddRow("01011112222"))
	mock.ExpectExec(`UPDATE WEO_MEMBER`).
		WithArgs("홍길동", "01011112222", "", "", "", "", "", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.UpdateMemberProfile(42, model.AdminMemberProfileUpdate{USRName: "홍길동"}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminUpdateMemberProfileRejectsClaimedPhoneAndRollsBack(t *testing.T) {
	repo, mock := newAdminMemberRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_PHONE FROM WEO_MEMBER`).WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_PHONE"}).AddRow("01011112222"))
	expectPhoneClaimsApplied(mock)
	mock.ExpectExec(`UPDATE AUTH_PHONE_CLAIM`).WithArgs("01033334444", 42).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate phone"})
	mock.ExpectRollback()

	err := repo.UpdateMemberProfile(42, model.AdminMemberProfileUpdate{USRName: "홍길동", USRPhone: "01033334444"})
	if !errors.Is(err, ErrPhoneAlreadyClaimed) {
		t.Fatalf("error = %v, want ErrPhoneAlreadyClaimed", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminUpdateMemberProfileReportsMissingMember(t *testing.T) {
	repo, mock := newAdminMemberRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_PHONE FROM WEO_MEMBER`).WithArgs(404).
		WillReturnRows(sqlmock.NewRows([]string{"USR_PHONE"}))
	mock.ExpectRollback()

	err := repo.UpdateMemberProfile(404, model.AdminMemberProfileUpdate{USRName: "홍길동"})
	if !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("error = %v, want ErrMemberNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminMemberListAppliesDepartmentAndJoinDateFilters(t *testing.T) {
	repo, mock := newAdminMemberRepoMock(t)
	filter := model.AdminMemberFilter{FN: "30", Dept: "영어", RegFrom: "2026-01-01", RegTo: "2026-01-31"}
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM WEO_MEMBER WHERE USR_FN = \? AND USR_DEPT = \? AND REG_DATE >= \? AND REG_DATE < DATE_ADD\(\?, INTERVAL 1 DAY\)`).
		WithArgs("30", "영어", "2026-01-01", "2026-01-31").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))
	mock.ExpectQuery(`SELECT USR_SEQ, USR_ID, USR_NAME`).
		WithArgs("30", "영어", "2026-01-01", "2026-01-31", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}))

	if _, _, err := repo.GetMembers(1, 20, filter); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
