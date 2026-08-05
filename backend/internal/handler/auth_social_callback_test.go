package handler

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

func TestSocialCallbackCreatesCanonicalContinuationWithoutEmailAutoLink(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{
		JWT:    config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour},
		Server: config.ServerConfig{AllowedOrigin: "https://example.invalid"},
	}
	cacheStore := cache.New(time.Minute, time.Minute)
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, handlerFakeKakaoClient{}, zerolog.Nop())
	handler := NewAuthHandler(svc, nil, nil, nil, cacheStore, cfg, zerolog.Nop())

	mock.ExpectQuery(regexp.QuoteMeta("WHERE s.NMS_GATE = ? AND s.NMS_ID = ? AND s.NMS_STATUS = 'ACTIVE'")).
		WithArgs("KT", "provider-subject").
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD")).
		WithArgs("KT", "provider-subject", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_SOCIAL_LINK_CONTINUATION")).
		WithArgs(sqlmock.AnyArg(), "KT", "provider-subject", "prefill@example.invalid", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	request := httptest.NewRequest(http.MethodGet, "/api/auth/kakao/callback", nil)
	recorder := httptest.NewRecorder()
	handler.handleSocialCallback(recorder, request, "KT", service.KakaoUserInfo{
		KakaoID: "provider-subject",
		Email:   "prefill@example.invalid",
	})

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	location := recorder.Header().Get("Location")
	if !strings.HasPrefix(location, "https://example.invalid/login/link?token=") {
		t.Fatalf("location = %q", location)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteSocialLoginUsesCentralPolicyForPendingPrincipal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{
		JWT:    config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour},
		Server: config.ServerConfig{AllowedOrigin: "https://example.invalid"},
	}
	cacheStore := cache.New(time.Minute, time.Minute)
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, handlerFakeKakaoClient{}, zerolog.Nop())
	handler := NewAuthHandler(svc, nil, nil, nil, cacheStore, cfg, zerolog.Nop())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "BBB", nil,
			"pending", nil, "10", "International", nil, time.Now(), nil))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE WEO_MEMBER")).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO WEO_MEMBER_LOG")).
		WithArgs(42, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	request := httptest.NewRequest(http.MethodGet, "/api/auth/kakao/callback", nil)
	recorder := httptest.NewRecorder()
	handler.completeSocialLogin(recorder, request, "KT", &model.User{
		USRSeq: 42, USRName: "Member", USRStatus: "BBB",
	}, "fixture-provider-token")

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != "https://example.invalid/" {
		t.Fatalf("location = %q", location)
	}
	if len(recorder.Result().Cookies()) != 6 {
		t.Fatalf("cookie count = %d", len(recorder.Result().Cookies()))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteSocialLoginMapsWithdrawnPrincipalToForbidden(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{
		JWT:    config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour},
		Server: config.ServerConfig{AllowedOrigin: "https://example.invalid"},
	}
	cacheStore := cache.New(time.Minute, time.Minute)
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, handlerFakeKakaoClient{}, zerolog.Nop())
	handler := NewAuthHandler(svc, nil, nil, nil, cacheStore, cfg, zerolog.Nop())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "AAA", nil,
			"approved", nil, "10", "International", nil, time.Now(), time.Now()))

	request := httptest.NewRequest(http.MethodGet, "/api/auth/kakao/callback", nil)
	recorder := httptest.NewRecorder()
	handler.completeSocialLogin(recorder, request, "KT", &model.User{
		USRSeq: 42, USRName: "Member", USRStatus: "CCC",
	}, "fixture-provider-token")

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"ACCOUNT_WITHDRAWN"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("cookies = %#v", recorder.Result().Cookies())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
