package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TestConsumeAppleChallengeIsSingleUse(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT NONCE_HASH, EXPIRES_AT, CONSUMED_AT`).
		WithArgs("challenge").
		WillReturnRows(sqlmock.NewRows([]string{"NONCE_HASH", "EXPIRES_AT", "CONSUMED_AT"}).
			AddRow("nonce-hash", now.Add(time.Minute), nil))
	mock.ExpectExec(`UPDATE ALUMNI_APPLE_NONCE_CHALLENGE`).
		WithArgs("challenge").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	nonceHash, err := repo.ConsumeAppleChallenge("challenge")
	if err != nil || nonceHash != "nonce-hash" {
		t.Fatalf("first consume = %q, %v", nonceHash, err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT NONCE_HASH, EXPIRES_AT, CONSUMED_AT`).
		WithArgs("challenge").
		WillReturnRows(sqlmock.NewRows([]string{"NONCE_HASH", "EXPIRES_AT", "CONSUMED_AT"}).
			AddRow("nonce-hash", now.Add(time.Minute), now))
	mock.ExpectRollback()

	_, err = repo.ConsumeAppleChallenge("challenge")
	if !errors.Is(err, ErrChallengeInvalid) {
		t.Fatalf("replay error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshReplayRevokesSessionFamily(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT MRT_SID, EXPIRES_AT, CONSUMED_AT, REVOKED_AT`).
		WithArgs("old-jti", 42).
		WillReturnRows(sqlmock.NewRows([]string{"MRT_SID", "EXPIRES_AT", "CONSUMED_AT", "REVOKED_AT"}).
			AddRow("family", now.Add(time.Hour), now, nil))
	mock.ExpectExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(42, "family").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	err = repo.RotateMobileRefreshToken(42, "family", "old-jti", "new-jti", now.Add(time.Hour))
	if !errors.Is(err, ErrRefreshTokenReplay) {
		t.Fatalf("rotation error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAppleAuthorizationCodeReplayIsRejected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`INSERT INTO ALUMNI_APPLE_CODE_REPLAY`).
		WithArgs("code-hash").
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := repo.ConsumeAppleAuthorizationCode("code-hash"); err != nil {
		t.Fatalf("first consume: %v", err)
	}

	mock.ExpectExec(`INSERT INTO ALUMNI_APPLE_CODE_REPLAY`).
		WithArgs("code-hash").
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate"})
	if err := repo.ConsumeAppleAuthorizationCode("code-hash"); !errors.Is(err, ErrAuthorizationReplay) {
		t.Fatalf("replay error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLogoutScopesCurrentSessionAndAllSessionsSeparately(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`WHERE USR_SEQ = \? AND MRT_SID = \?`).
		WithArgs(42, "current-device").
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.RevokeMobileSession(42, "current-device"); err != nil {
		t.Fatal(err)
	}

	mock.ExpectExec(`WHERE USR_SEQ = \?`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 3))
	if err := repo.RevokeAllMobileSessions(42); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountConnectionsReturnsOnlyTypedProvidersAndPasswordState(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT CASE WHEN IFNULL\(USR_PWD, ''\)`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"HAS_PASSWORD"}).AddRow(true))
	mock.ExpectQuery(`SELECT NMS_GATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}).
			AddRow("KT").
			AddRow("UNKNOWN").
			AddRow("AP"))

	connections, err := repo.GetAccountConnections(42)
	if err != nil {
		t.Fatal(err)
	}
	if !connections.HasPassword {
		t.Fatal("password state was not returned")
	}
	expectedProviders := []model.SocialProvider{
		model.SocialProviderApple,
		model.SocialProviderKakao,
	}
	if len(connections.Providers) != len(expectedProviders) {
		t.Fatalf("providers = %#v", connections.Providers)
	}
	for index, provider := range expectedProviders {
		if connections.Providers[index] != provider {
			t.Fatalf("providers = %#v, want %#v", connections.Providers, expectedProviders)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
