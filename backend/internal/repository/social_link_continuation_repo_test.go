package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestInsertSocialLinkContinuationPersistsSubjectGuardAndTokenAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	expiresAt := time.Now().Add(time.Minute)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD[\s\S]+ON DUPLICATE KEY UPDATE`).
		WithArgs("KT", "provider-subject", expiresAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_LINK_CONTINUATION`).
		WithArgs("new-token-hash", "KT", "provider-subject", "provider@example.com", expiresAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := repo.InsertSocialLinkContinuation(
		"new-token-hash", "KT", "provider-subject", "provider@example.com", expiresAt,
	); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteSocialLinkContinuationReauthenticatesAttachesAndConsumesAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT SLC_PROVIDER, SLC_SUBJECT, SLC_EMAIL, SLC_STATUS, SLC_EXPIRES_AT`).
		WithArgs("fixture-token-hash").
		WillReturnRows(sqlmock.NewRows([]string{
			"SLC_PROVIDER", "SLC_SUBJECT", "SLC_EMAIL", "SLC_STATUS", "SLC_EXPIRES_AT",
		}).AddRow("KT", "fixture-provider-subject", "provider@example.com", "READY", time.Now().Add(time.Minute)))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD[\s\S]+ON DUPLICATE KEY UPDATE`).
		WithArgs("KT", "fixture-provider-subject", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT SLR_FAILED_ATTEMPTS, SLR_LOCKED_AT[\s\S]+FOR UPDATE`).
		WithArgs("KT", "fixture-provider-subject").
		WillReturnRows(sqlmock.NewRows([]string{"SLR_FAILED_ATTEMPTS", "SLR_LOCKED_AT"}).AddRow(0, nil))
	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\)`).
		WithArgs("member@example.com", "fixture-password-hash").
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", nil, "10", "member@example.com", nil, nil))
	mock.ExpectQuery(`SELECT USR_SEQ\s+FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "fixture-provider-subject").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}))
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT", "fixture-provider-subject", "provider@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_LINK_CONTINUATION`).
		WithArgs("fixture-token-hash").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	user, err := repo.CompleteSocialLinkContinuation(
		"fixture-token-hash",
		"member@example.com",
		"fixture-password-hash",
	)
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

func TestCompleteSocialLinkContinuationLocksOnFifthReauthFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT SLC_PROVIDER, SLC_SUBJECT, SLC_EMAIL, SLC_STATUS, SLC_EXPIRES_AT`).
		WithArgs("fixture-token-hash").
		WillReturnRows(sqlmock.NewRows([]string{
			"SLC_PROVIDER", "SLC_SUBJECT", "SLC_EMAIL", "SLC_STATUS", "SLC_EXPIRES_AT",
		}).AddRow("KT", "fixture-provider-subject", "provider@example.com", "READY", time.Now().Add(time.Minute)))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD[\s\S]+ON DUPLICATE KEY UPDATE`).
		WithArgs("KT", "fixture-provider-subject", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT SLR_FAILED_ATTEMPTS, SLR_LOCKED_AT[\s\S]+FOR UPDATE`).
		WithArgs("KT", "fixture-provider-subject").
		WillReturnRows(sqlmock.NewRows([]string{"SLR_FAILED_ATTEMPTS", "SLR_LOCKED_AT"}).AddRow(4, nil))
	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\)`).
		WithArgs("member@example.com", "wrong-password-hash").
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}))
	mock.ExpectExec(`SET SLR_FAILED_ATTEMPTS = SLR_FAILED_ATTEMPTS \+ 1`).
		WithArgs(5, "KT", "fixture-provider-subject").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	user, err := repo.CompleteSocialLinkContinuation("fixture-token-hash", "member@example.com", "wrong-password-hash")
	if !errors.Is(err, ErrSocialLinkReauthLocked) {
		t.Fatalf("error = %v, want ErrSocialLinkReauthLocked", err)
	}
	if user != nil {
		t.Fatalf("user = %#v, want nil", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteSocialLinkContinuationSerializesFailuresByProviderSubjectGuard(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT SLC_PROVIDER, SLC_SUBJECT[\s\S]+FOR UPDATE`).
		WithArgs("fixture-token-hash").
		WillReturnRows(sqlmock.NewRows([]string{
			"SLC_PROVIDER", "SLC_SUBJECT", "SLC_EMAIL", "SLC_STATUS", "SLC_EXPIRES_AT",
		}).AddRow("KT", "fixture-provider-subject", nil, "READY", time.Now().Add(time.Minute)))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD[\s\S]+ON DUPLICATE KEY UPDATE`).
		WithArgs("KT", "fixture-provider-subject", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT SLR_FAILED_ATTEMPTS, SLR_LOCKED_AT[\s\S]+FOR UPDATE`).
		WithArgs("KT", "fixture-provider-subject").
		WillReturnRows(sqlmock.NewRows([]string{"SLR_FAILED_ATTEMPTS", "SLR_LOCKED_AT"}).AddRow(0, nil))
	mock.ExpectQuery(`SELECT USR_SEQ, USR_ID[\s\S]+FROM WEO_MEMBER`).
		WithArgs("member@example.com", "hashed-password").
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_LINK_REAUTH_GUARD`).
		WithArgs(5, "KT", "fixture-provider-subject").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	_, err = repo.CompleteSocialLinkContinuation(
		"fixture-token-hash", "member@example.com", "hashed-password",
	)
	if !errors.Is(err, ErrSocialLinkReauth) {
		t.Fatalf("expected ErrSocialLinkReauth, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteSocialLinkContinuationRejectsLockedTokenBeforeCredentialQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT SLC_PROVIDER, SLC_SUBJECT, SLC_EMAIL, SLC_STATUS, SLC_EXPIRES_AT`).
		WithArgs("fixture-token-hash").
		WillReturnRows(sqlmock.NewRows([]string{
			"SLC_PROVIDER", "SLC_SUBJECT", "SLC_EMAIL", "SLC_STATUS", "SLC_EXPIRES_AT",
		}).AddRow("KT", "fixture-provider-subject", "provider@example.com", "READY", time.Now().Add(time.Minute)))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD[\s\S]+ON DUPLICATE KEY UPDATE`).
		WithArgs("KT", "fixture-provider-subject", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT SLR_FAILED_ATTEMPTS, SLR_LOCKED_AT[\s\S]+FOR UPDATE`).
		WithArgs("KT", "fixture-provider-subject").
		WillReturnRows(sqlmock.NewRows([]string{"SLR_FAILED_ATTEMPTS", "SLR_LOCKED_AT"}).AddRow(5, time.Now()))
	mock.ExpectRollback()

	user, err := repo.CompleteSocialLinkContinuation("fixture-token-hash", "member@example.com", "password-hash")
	if !errors.Is(err, ErrSocialLinkReauthLocked) {
		t.Fatalf("error = %v, want ErrSocialLinkReauthLocked", err)
	}
	if user != nil {
		t.Fatalf("user = %#v, want nil", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
