package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
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

func TestLogoutReturnsFailureWhenCurrentSessionRevokeFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	cacheStore := cache.New(time.Minute, time.Minute)
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, nil, zerolog.Nop())
	handler := NewAuthHandler(svc, service.NewMemberService(repo), nil, nil, cacheStore, cfg, zerolog.Nop())
	mock.ExpectExec(regexp.QuoteMeta("UPDATE ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(42, "current-family").
		WillReturnError(errors.New("fixture revoke failure"))

	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{
		USRSeq:    42,
		SessionID: "current-family",
	}))
	recorder := httptest.NewRecorder()

	handler.Logout(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("cookies were cleared after failed revoke: %#v", recorder.Result().Cookies())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLogoutAllReturnsFailureWhenSessionRevokeFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	cacheStore := cache.New(time.Minute, time.Minute)
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, nil, zerolog.Nop())
	handler := NewAuthHandler(svc, service.NewMemberService(repo), nil, nil, cacheStore, cfg, zerolog.Nop())
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM WEO_MEMBER_LOG WHERE USR_SEQ = ?")).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`SET MRT_REVOKED_AT = COALESCE\(MRT_REVOKED_AT, NOW\(\)\),\s+REVOKED_AT = COALESCE\(REVOKED_AT, NOW\(\)\)`).
		WithArgs(42).
		WillReturnError(errors.New("fixture revoke-all failure"))

	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout/all", nil)
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42}))
	recorder := httptest.NewRecorder()

	handler.LogoutAll(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("cookies were cleared after failed revoke-all: %#v", recorder.Result().Cookies())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
