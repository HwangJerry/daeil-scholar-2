package service

import (
	"errors"
	"net/http/httptest"
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

func TestLoginWithBridgeRejectsFreshlyIneligiblePrincipalBeforeCookieOrSessionWrite(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	svc := NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "AAA", nil,
			"approved", nil, "10", "International", nil, time.Now(), time.Now()))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/api/auth/login", nil)
	err = svc.LoginWithBridge(&model.User{
		USRSeq:    42,
		USRID:     "member",
		USRName:   "Member",
		USRStatus: "CCC",
	}, recorder, request)

	if !errors.Is(err, ErrLoginWithdrawn) {
		t.Fatalf("LoginWithBridge error = %v, want ErrLoginWithdrawn", err)
	}
	if cookies := recorder.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("issued %d auth cookies for ineligible principal", len(cookies))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginWithBridgeDoesNotInsertSessionLogWhenLastLoginUpdateFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	svc := NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "CCC", nil,
			"approved", nil, "10", "International", nil, time.Now(), time.Now()))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE WEO_MEMBER")).
		WithArgs(42).
		WillReturnError(errors.New("fixture last-login failure"))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/api/auth/login", nil)
	err = svc.LoginWithBridge(&model.User{
		USRSeq:    42,
		USRID:     "member",
		USRName:   "Member",
		USRStatus: "CCC",
	}, recorder, request)

	if err == nil {
		t.Fatal("LoginWithBridge error = nil")
	}
	if cookies := recorder.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("issued %d auth cookies after last-login failure", len(cookies))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginWithBridgeReturnsErrorWithoutCookiesWhenSessionInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	svc := NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "CCC", nil,
			"approved", nil, "10", "International", nil, time.Now(), time.Now()))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE WEO_MEMBER")).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	sessionInsertFailure := errors.New("fixture session insert failure")
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO WEO_MEMBER_LOG")).
		WithArgs(42, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(sessionInsertFailure)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/api/auth/login", nil)
	err = svc.LoginWithBridge(&model.User{
		USRSeq: 42, USRID: "member", USRName: "Member", USRStatus: "CCC",
	}, recorder, request)

	if !errors.Is(err, sessionInsertFailure) {
		t.Fatalf("LoginWithBridge error = %v, want session insert failure", err)
	}
	if cookies := recorder.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("issued %d auth cookies after session insert failure", len(cookies))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
