package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

func TestProtectedBusinessRouteRejectsPendingPrincipal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	authService := service.NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())

	for range 2 {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
			WithArgs(42).
			WillReturnRows(sqlmock.NewRows([]string{
				"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
				"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
			}).AddRow(42, "member", "Member", "member@example.com", "BBB", nil,
				"pending", nil, "10", "International", nil, time.Now(), nil))
	}

	token, err := authService.GenerateJWT(&model.AuthUser{USRSeq: 42, USRName: "Member", USRStatus: "BBB"})
	if err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	registerAuthRoutes(router, handlers{}, authService)
	request := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	request.AddCookie(&http.Cookie{Name: "alumni_token", Value: token})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminRouteRejectsPendingOperatorPrincipal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	authService := service.NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())

	for range 2 {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
			WithArgs(42).
			WillReturnRows(sqlmock.NewRows([]string{
				"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
				"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
			}).AddRow(42, "member", "Member", "member@example.com", "BBB", "operator",
				"pending", nil, "10", "International", nil, time.Now(), nil))
	}

	token, err := authService.GenerateJWT(&model.AuthUser{USRSeq: 42, USRName: "Member", USRStatus: "BBB"})
	if err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	registerAdminRoutes(router, handlers{}, authService, cfg)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	request.AddCookie(&http.Cookie{Name: "alumni_token", Value: token})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSocialLinkAliasesUseLoginRateLimiter(t *testing.T) {
	for _, path := range []string{"/api/auth/social/link", "/api/auth/kakao/link"} {
		t.Run(path, func(t *testing.T) {
			cacheStore := cache.New(time.Minute, time.Minute)
			cacheStore.Set("login_attempts:192.0.2.1", 10, time.Minute)
			router := chi.NewRouter()
			registerPublicRoutes(router, handlers{}, cacheStore)

			request := httptest.NewRequest(http.MethodPost, path, nil)
			request.RemoteAddr = "192.0.2.1:1234"
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusTooManyRequests {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTooManyRequests)
			}
		})
	}
}
