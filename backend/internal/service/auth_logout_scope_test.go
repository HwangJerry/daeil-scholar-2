package service

import (
	"database/sql"
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

func TestLogoutCurrentRevokesOnlyCurrentMobileSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := newLogoutScopeService(db)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(42, "current-family").
		WillReturnResult(sqlmock.NewResult(0, 2))

	recorder := httptest.NewRecorder()
	svc.LogoutCurrent(recorder, &model.AuthUser{USRSeq: 42, SessionID: "current-family"})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	assertAuthCookiesCleared(t, recorder)
}

func TestLogoutCurrentDeletesOnlyCurrentLegacySession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := newLogoutScopeService(db)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM WEO_MEMBER_LOG WHERE USR_SEQ = ? AND SESSIONID = ?")).
		WithArgs(42, "current-legacy-session").
		WillReturnResult(sqlmock.NewResult(0, 1))

	recorder := httptest.NewRecorder()
	svc.LogoutCurrent(recorder, &model.AuthUser{
		USRSeq:          42,
		LegacySessionID: "current-legacy-session",
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	assertAuthCookiesCleared(t, recorder)
}

func TestLogoutCurrentReturnsErrorBeforeClearingCookiesWhenMobileRevokeFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := newLogoutScopeService(db)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(42, "current-family").
		WillReturnError(errors.New("fixture revoke failure"))

	recorder := httptest.NewRecorder()
	err = svc.LogoutCurrent(recorder, &model.AuthUser{USRSeq: 42, SessionID: "current-family"})

	if err == nil {
		t.Fatal("LogoutCurrent error = nil")
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("cookies were cleared before revoke succeeded: %#v", recorder.Result().Cookies())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLogoutAllRevokesLegacyAndAllMobileSessions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := newLogoutScopeService(db)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM WEO_MEMBER_LOG WHERE USR_SEQ = ?")).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`SET MRT_REVOKED_AT = COALESCE\(MRT_REVOKED_AT, NOW\(\)\),\s+REVOKED_AT = COALESCE\(REVOKED_AT, NOW\(\)\)`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 3))

	recorder := httptest.NewRecorder()
	svc.LogoutAll(recorder, 42)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	assertAuthCookiesCleared(t, recorder)
}

func TestLogoutAllReturnsErrorBeforeClearingCookiesWhenMobileRevokeFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := newLogoutScopeService(db)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM WEO_MEMBER_LOG WHERE USR_SEQ = ?")).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`SET MRT_REVOKED_AT = COALESCE\(MRT_REVOKED_AT, NOW\(\)\),\s+REVOKED_AT = COALESCE\(REVOKED_AT, NOW\(\)\)`).
		WithArgs(42).
		WillReturnError(errors.New("fixture revoke-all failure"))

	recorder := httptest.NewRecorder()
	err = svc.LogoutAll(recorder, 42)

	if err == nil {
		t.Fatal("LogoutAll error = nil")
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("cookies were cleared before revoke-all succeeded: %#v", recorder.Result().Cookies())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLogoutCurrentRevokesMobileAndCompanionLegacySessions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := newLogoutScopeService(db)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(42, "current-family").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM WEO_MEMBER_LOG WHERE USR_SEQ = ? AND SESSIONID = ?")).
		WithArgs(42, "current-legacy-session").
		WillReturnResult(sqlmock.NewResult(0, 1))

	recorder := httptest.NewRecorder()
	err = svc.LogoutCurrent(recorder, &model.AuthUser{
		USRSeq:          42,
		SessionID:       "current-family",
		LegacySessionID: "current-legacy-session",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	assertAuthCookiesCleared(t, recorder)
}

func TestLogoutCurrentPreservesUserWideKakaoCache(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := newLogoutScopeService(db)
	svc.cache.Set("kakao_token:42", 1, time.Minute)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(42, "current-family").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.LogoutCurrent(httptest.NewRecorder(), &model.AuthUser{USRSeq: 42, SessionID: "current-family"}); err != nil {
		t.Fatal(err)
	}
	if _, found := svc.cache.Get("kakao_token:42"); !found {
		t.Fatal("current logout removed user-wide Kakao provider cache")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func newLogoutScopeService(db *sql.DB) *AuthService {
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	return NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
}

func assertAuthCookiesCleared(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	cleared := map[string]bool{}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.MaxAge < 0 {
			cleared[cookie.Name] = true
		}
	}
	for _, name := range []string{"alumni_token", "DDusrSession_id", "DDusrSEQ", "DDusrID", "DDusrNAME", "DDusrSTATUS"} {
		if !cleared[name] {
			t.Fatalf("cookie %q was not cleared", name)
		}
	}
}
