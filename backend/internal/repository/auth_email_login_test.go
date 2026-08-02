// auth_email_login_test.go — Verifies canonical email/password lookup fails closed on ambiguity.
package repository_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
)

func TestFindMemberByEmailAndPwdAnyUsesCanonicalEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	password := service.MysqlNativePassword("correct-password")

	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\) AND USR_PWD = \?`).
		WithArgs(" Member@Example.com ", password).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE",
			"USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "legacy-id", "Member", "BBB", nil, nil, "member@example.com", nil, nil))

	user, err := repo.FindMemberByEmailAndPwdAny(" Member@Example.com ", password)
	if err != nil {
		t.Fatal(err)
	}
	if user == nil || user.USRSeq != 42 {
		t.Fatalf("user = %#v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindMemberByEmailAndPwdAnyRejectsAmbiguousMatches(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	password := service.MysqlNativePassword("shared-password")

	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\) AND USR_PWD = \?`).
		WithArgs("duplicate@example.com", password).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE",
			"USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).
			AddRow(42, "first", "First", "CCC", nil, nil, "duplicate@example.com", nil, nil).
			AddRow(43, "second", "Second", "CCC", nil, nil, "duplicate@example.com", nil, nil))

	user, err := repo.FindMemberByEmailAndPwdAny("duplicate@example.com", password)
	if err != nil {
		t.Fatal(err)
	}
	if user != nil {
		t.Fatalf("ambiguous email returned user %#v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
