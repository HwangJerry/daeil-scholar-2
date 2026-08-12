package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func canonicalReplacementFixture() model.PasswordCredential {
	parameters := "m=19456,t=2,p=1"
	return model.PasswordCredential{
		Provider: model.IdentityProviderLocalUsername, Algorithm: model.PasswordAlgorithmArgon2id,
		ParametersText: &parameters, PasswordHash: "new-canonical-hash", Status: model.PasswordCredentialStatusActive,
	}
}

func activePasswordIdentityRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"IDENTITY_ID", "PROVIDER", "IDENTITY_STATUS", "ALGORITHM", "PARAMETERS_TEXT", "PASSWORD_HASH", "CREDENTIAL_STATUS",
	}).AddRow(int64(101), "LOCAL_USERNAME", "ACTIVE", "ARGON2ID", "m=19456,t=2,p=1", "old-canonical-hash", "ACTIVE")
}

func mixedActiveAndDisabledPasswordIdentityRows() *sqlmock.Rows {
	return activePasswordIdentityRows().AddRow(
		int64(102), "EMAIL", "ACTIVE", "ARGON2ID", "m=19456,t=2,p=1", "disabled-hash", "DISABLED",
	)
}

func TestPasswordMutationRepositoryChangesCanonicalAndLegacyAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewPasswordMutationRepository(sqlx.NewDb(db, "sqlmock"))
	replacement := canonicalReplacementFixture()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT IFNULL\(USR_PWD`).WithArgs(42).WillReturnRows(sqlmock.NewRows([]string{"USR_PWD"}).AddRow("legacy-old"))
	mock.ExpectQuery(`FROM AUTH_IDENTITY i`).WithArgs(42).WillReturnRows(activePasswordIdentityRows())
	mock.ExpectExec(`INSERT INTO AUTH_PASSWORD_CREDENTIAL`).WithArgs(int64(101), "LOCAL_USERNAME", replacement.Algorithm, replacement.ParametersText, replacement.PasswordHash).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE WEO_MEMBER SET USR_PWD`).WithArgs("legacy-new", 42).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repo.ChangePasswordAtomically(42, "legacy-submitted", "legacy-new", replacement, func(credential model.PasswordCredential) (bool, error) {
		return credential.PasswordHash == "old-canonical-hash", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPasswordMutationRepositoryRejectsWrongCanonicalWithoutLegacyFallback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewPasswordMutationRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT IFNULL\(USR_PWD`).WillReturnRows(sqlmock.NewRows([]string{"USR_PWD"}).AddRow("legacy-submitted"))
	mock.ExpectQuery(`FROM AUTH_IDENTITY i`).WillReturnRows(activePasswordIdentityRows())
	mock.ExpectRollback()

	err = repo.ChangePasswordAtomically(42, "legacy-submitted", "legacy-new", canonicalReplacementFixture(), func(model.PasswordCredential) (bool, error) {
		return false, nil
	})
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("ChangePasswordAtomically() error = %v, want ErrPasswordMismatch", err)
	}
}

func TestPasswordMutationRepositoryRejectsAccountWithoutPasswordIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewPasswordMutationRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT IFNULL\(USR_PWD`).WillReturnRows(sqlmock.NewRows([]string{"USR_PWD"}).AddRow("legacy-submitted"))
	mock.ExpectQuery(`FROM AUTH_IDENTITY i`).WillReturnRows(sqlmock.NewRows([]string{
		"IDENTITY_ID", "PROVIDER", "IDENTITY_STATUS", "ALGORITHM", "PARAMETERS_TEXT", "PASSWORD_HASH", "CREDENTIAL_STATUS",
	}))
	mock.ExpectRollback()

	err = repo.ChangePasswordAtomically(42, "legacy-submitted", "legacy-new", canonicalReplacementFixture(), func(model.PasswordCredential) (bool, error) {
		return true, nil
	})
	if !errors.Is(err, ErrPasswordIdentityMissing) {
		t.Fatalf("ChangePasswordAtomically() error = %v, want ErrPasswordIdentityMissing", err)
	}
}

func TestPasswordMutationRepositoryDoesNotReactivateDisabledCredential(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewPasswordMutationRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT IFNULL\(USR_PWD`).WillReturnRows(sqlmock.NewRows([]string{"USR_PWD"}).AddRow("legacy-submitted"))
	mock.ExpectQuery(`FROM AUTH_IDENTITY i`).WillReturnRows(mixedActiveAndDisabledPasswordIdentityRows())
	mock.ExpectRollback()

	err = repo.ChangePasswordAtomically(42, "legacy-submitted", "legacy-new", canonicalReplacementFixture(), func(model.PasswordCredential) (bool, error) {
		return true, nil
	})
	if !errors.Is(err, ErrPasswordCredentialDisabled) {
		t.Fatalf("ChangePasswordAtomically() error = %v, want ErrPasswordCredentialDisabled", err)
	}
}
