package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func passwordSignupFixture() (model.RegisterRequest, model.PasswordCredential) {
	parameters := "m=19456,t=2,p=1"
	return model.RegisterRequest{
			UsrID: "new-member", Name: "Member", Phone: "01012345678", Password: "not-persisted", Tags: []string{"alumni"},
		}, model.PasswordCredential{
			Provider: model.IdentityProviderLocalUsername, Algorithm: model.PasswordAlgorithmArgon2id,
			ParametersText: &parameters, PasswordHash: "canonical-hash", Status: model.PasswordCredentialStatusActive,
		}
}

func TestSignupRepositoryCreatesPasswordAccountAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSignupRepository(sqlx.NewDb(db, "sqlmock"))
	request, credential := passwordSignupFixture()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO AUTH_PHONE_CLAIM`).WithArgs(request.Phone, 42).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO AUTH_ACCOUNT_STATE`).WithArgs(42).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).WithArgs(42, model.VerificationUnsubmitted).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO AUTH_IDENTITY`).WithArgs(42, string(model.IdentityProviderLocalUsername), request.UsrID).WillReturnResult(sqlmock.NewResult(101, 1))
	mock.ExpectExec(`INSERT INTO AUTH_PASSWORD_CREDENTIAL`).WithArgs(int64(101), string(model.IdentityProviderLocalUsername), credential.Algorithm, credential.ParametersText, credential.PasswordHash).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_USER_TAG`).WithArgs(42, "alumni", 0).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	accountSeq, err := repo.CreatePasswordAccount(request, "legacy-hash", credential)
	if err != nil {
		t.Fatal(err)
	}
	if accountSeq != 42 {
		t.Fatalf("accountSeq = %d, want 42", accountSeq)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSignupRepositoryRollsBackAfterCanonicalWriteFailures(t *testing.T) {
	tests := []struct {
		name        string
		failPattern string
	}{
		{name: "phone_claim", failPattern: `INSERT INTO AUTH_PHONE_CLAIM`},
		{name: "account_state", failPattern: `INSERT INTO AUTH_ACCOUNT_STATE`},
		{name: "identity", failPattern: `INSERT INTO AUTH_IDENTITY`},
		{name: "credential", failPattern: `INSERT INTO AUTH_PASSWORD_CREDENTIAL`},
		{name: "tag", failPattern: `INSERT INTO ALUMNI_USER_TAG`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := NewSignupRepository(sqlx.NewDb(db, "sqlmock"))
			request, credential := passwordSignupFixture()

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO WEO_MEMBER`).WillReturnResult(sqlmock.NewResult(42, 1))
			if test.name == "phone_claim" {
				mock.ExpectExec(test.failPattern).WillReturnError(errors.New("injected signup failure"))
			} else {
				mock.ExpectExec(`INSERT INTO AUTH_PHONE_CLAIM`).WillReturnResult(sqlmock.NewResult(1, 1))
			}
			if test.name == "account_state" {
				mock.ExpectExec(test.failPattern).WillReturnError(errors.New("injected signup failure"))
			} else if test.name != "phone_claim" {
				mock.ExpectExec(`INSERT INTO AUTH_ACCOUNT_STATE`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).WillReturnResult(sqlmock.NewResult(1, 1))
				if test.name == "identity" {
					mock.ExpectExec(test.failPattern).WillReturnError(errors.New("injected signup failure"))
				} else {
					mock.ExpectExec(`INSERT INTO AUTH_IDENTITY`).WillReturnResult(sqlmock.NewResult(101, 1))
					if test.name == "credential" {
						mock.ExpectExec(test.failPattern).WillReturnError(errors.New("injected signup failure"))
					} else {
						mock.ExpectExec(`INSERT INTO AUTH_PASSWORD_CREDENTIAL`).WillReturnResult(sqlmock.NewResult(1, 1))
						mock.ExpectExec(test.failPattern).WillReturnError(errors.New("injected signup failure"))
					}
				}
			}
			mock.ExpectRollback()

			if _, err := repo.CreatePasswordAccount(request, "legacy-hash", credential); err == nil {
				t.Fatal("CreatePasswordAccount() error = nil")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
