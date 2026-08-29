// auth_logout_handler_test.go — Verifies canonical logout status and revocation scope.
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

func TestLogoutReturnsNoContentAfterRevokingCurrentSession(t *testing.T) {
	handler, mock, cleanup := newLogoutAuthHandlerForTest(t)
	defer cleanup()
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_LOG WHERE SESSIONID`).
		WithArgs("legacy-session").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`WHERE USR_SEQ = \? AND MRT_SID = \?`).
		WithArgs(42, "mobile-session").
		WillReturnResult(sqlmock.NewResult(0, 1))

	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: "DDusrSession_id", Value: "legacy-session"})
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{
		USRSeq:    42,
		SessionID: "mobile-session",
	}))
	recorder := httptest.NewRecorder()

	handler.Logout(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("logout body must be empty: %q", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLogoutAllReturnsNoContentAndRemovesDeviceAssociations(t *testing.T) {
	handler, mock, cleanup := newLogoutAuthHandlerForTest(t)
	defer cleanup()
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_LOG WHERE USR_SEQ`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 1))

	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout/all", nil)
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42}))
	recorder := httptest.NewRecorder()

	handler.LogoutAll(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("logout-all body must be empty: %q", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func newLogoutAuthHandlerForTest(t *testing.T) (*AuthHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	cfg := &config.Config{
		Server: config.ServerConfig{AllowedOrigin: "http://localhost"},
		JWT: config.JWTConfig{
			Secret:          "test-secret",
			MaxAge:          time.Hour,
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 30 * 24 * time.Hour,
		},
	}
	authRepo := repository.NewAuthRepository(sqlxDB)
	auth := service.NewAuthService(
		authRepo,
		repository.NewSessionRepository(sqlxDB),
		cfg,
		cache.New(time.Minute, time.Minute),
		zerolog.Nop(),
	)
	return &AuthHandler{service: auth, logger: zerolog.Nop()}, mock, func() { _ = db.Close() }
}
