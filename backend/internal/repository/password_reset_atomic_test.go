package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestPasswordResetRepositoryConfirmsResetAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewPasswordResetRepository(sqlx.NewDb(db, "sqlmock"))
	replacement := canonicalReplacementFixture()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*APR_TOKEN IN`).WithArgs("token-hash", "legacy-token").WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(42))
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*FROM WEO_MEMBER[\s\S]*FOR UPDATE`).WithArgs(42).WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(42))
	mock.ExpectQuery(`SELECT COUNT\(\*\)[\s\S]*APR_TOKEN IN`).WithArgs(42, "token-hash", "legacy-token").WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
	mock.ExpectQuery(`FROM AUTH_IDENTITY i`).WithArgs(42).WillReturnRows(activePasswordIdentityRows())
	mock.ExpectExec(`INSERT INTO AUTH_PASSWORD_CREDENTIAL`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE WEO_MEMBER SET USR_PWD`).WithArgs("legacy-new", 42).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_PASSWORD_RESET`).WithArgs(42).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN`).WithArgs(42).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`UPDATE AUTH_SESSION_FAMILY`).WithArgs(42).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repo.ConfirmResetAtomically("token-hash", "legacy-token", "legacy-new", replacement)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPasswordResetRepositoryRollsBackWhenCredentialWriteFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewPasswordResetRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*APR_TOKEN IN`).WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(42))
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*FROM WEO_MEMBER[\s\S]*FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(42))
	mock.ExpectQuery(`SELECT COUNT\(\*\)[\s\S]*APR_TOKEN IN`).WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
	mock.ExpectQuery(`FROM AUTH_IDENTITY i`).WillReturnRows(activePasswordIdentityRows())
	mock.ExpectExec(`INSERT INTO AUTH_PASSWORD_CREDENTIAL`).WillReturnError(errors.New("injected credential failure"))
	mock.ExpectRollback()

	if err := repo.ConfirmResetAtomically("token-hash", "legacy-token", "legacy-new", canonicalReplacementFixture()); err == nil {
		t.Fatal("ConfirmResetAtomically() error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPasswordResetRepositoryDoesNotReactivateDisabledCredential(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewPasswordResetRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*APR_TOKEN IN`).WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(42))
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*FROM WEO_MEMBER[\s\S]*FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(42))
	mock.ExpectQuery(`SELECT COUNT\(\*\)[\s\S]*APR_TOKEN IN`).WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
	mock.ExpectQuery(`FROM AUTH_IDENTITY i`).WillReturnRows(mixedActiveAndDisabledPasswordIdentityRows())
	mock.ExpectRollback()

	err = repo.ConfirmResetAtomically("token-hash", "legacy-token", "legacy-new", canonicalReplacementFixture())
	if !errors.Is(err, ErrPasswordCredentialDisabled) {
		t.Fatalf("ConfirmResetAtomically() error = %v, want ErrPasswordCredentialDisabled", err)
	}
}
