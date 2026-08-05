package service

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

func TestRotateMobileSessionRejectsFreshlyIneligiblePrincipalBeforeRotation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	svc := NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	refreshToken, _, _, err := svc.GenerateMobileRefreshJWT(&model.AuthUser{
		USRSeq:    42,
		USRName:   "Stale Member",
		USRStatus: "CCC",
	}, "mobile-family")
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", nil, "10", "member@example.com", nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "AAA", nil,
			"approved", nil, "10", "International", nil, time.Now(), time.Now()))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(42, "mobile-family").
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err = svc.RotateMobileSession(refreshToken)
	if !errors.Is(err, ErrLoginWithdrawn) {
		t.Fatalf("RotateMobileSession error = %v, want ErrLoginWithdrawn", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRotateMobileSessionReturnsRevokeFailureForFreshlyIneligiblePrincipal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	svc := NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	refreshToken, _, _, err := svc.GenerateMobileRefreshJWT(&model.AuthUser{
		USRSeq: 42, USRName: "Stale Member", USRStatus: "CCC",
	}, "mobile-family")
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", nil, "10", "member@example.com", nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "AAA", nil,
			"approved", nil, "10", "International", nil, time.Now(), time.Now()))
	revokeFailure := errors.New("fixture family revoke failure")
	mock.ExpectExec(regexp.QuoteMeta("UPDATE ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(42, "mobile-family").
		WillReturnError(revokeFailure)

	session, err := svc.RotateMobileSession(refreshToken)
	if !errors.Is(err, revokeFailure) {
		t.Fatalf("RotateMobileSession error = %v, want revoke failure", err)
	}
	if session != nil {
		t.Fatalf("session returned after revoke failure: %#v", session)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRotateMobileSessionReturnsNoSessionForConcurrentLoser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	svc := NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	refreshToken, oldJTI, _, err := svc.GenerateMobileRefreshJWT(&model.AuthUser{
		USRSeq:    42,
		USRName:   "Member",
		USRStatus: "CCC",
	}, "mobile-family")
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", nil, "10", "member@example.com", nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "CCC", nil,
			"approved", nil, "10", "International", nil, time.Now(), time.Now()))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT MRT_SID, EXPIRES_AT, CONSUMED_AT, REVOKED_AT, MRT_REVOKED_AT`).
		WithArgs(oldJTI, 42).
		WillReturnRows(sqlmock.NewRows([]string{
			"MRT_SID", "EXPIRES_AT", "CONSUMED_AT", "REVOKED_AT", "MRT_REVOKED_AT",
		}).AddRow("mobile-family", time.Now().Add(time.Hour), time.Now(), nil, nil))
	mock.ExpectExec(`SET MRT_REVOKED_AT = COALESCE\(MRT_REVOKED_AT, NOW\(\)\),\s+REVOKED_AT = COALESCE\(REVOKED_AT, NOW\(\)\)`).
		WithArgs(42, "mobile-family").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	session, err := svc.RotateMobileSession(refreshToken)
	if !errors.Is(err, repository.ErrRefreshTokenReplay) {
		t.Fatalf("RotateMobileSession error = %v", err)
	}
	if session != nil {
		t.Fatalf("concurrent loser received session: %#v", session)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
