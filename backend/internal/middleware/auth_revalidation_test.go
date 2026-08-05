package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
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

func TestAuthMiddlewareRejectsJWTCookieWhenCurrentAccountBecomesIneligible(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	svc := service.NewAuthService(repo, nil, &config.Config{JWT: config.JWTConfig{
		Secret: "fixture-secret",
		MaxAge: time.Hour,
	}}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	token, err := svc.GenerateJWT(&model.AuthUser{USRSeq: 42, USRStatus: "CCC"})
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(principalRows().AddRow(
			42, "member", "Member", "member@example.com", "AAA", nil,
			"approved", nil, nil, nil, nil, nil, nil,
		))

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	request.AddCookie(&http.Cookie{Name: jwtCookieName, Value: token})
	recorder := httptest.NewRecorder()

	AuthMiddleware(svc)(next).ServeHTTP(recorder, request)

	if called {
		t.Fatal("ineligible principal reached authenticated handler")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthMiddlewarePreservesCompanionLegacySessionForScopedLogout(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	svc := service.NewAuthService(repo, nil, &config.Config{JWT: config.JWTConfig{
		Secret: "fixture-secret",
		MaxAge: time.Hour,
	}}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	token, err := svc.GenerateJWT(&model.AuthUser{USRSeq: 42, USRStatus: "CCC"})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(principalRows().AddRow(
			42, "member", "Member", "member@example.com", "CCC", nil,
			"approved", nil, nil, nil, nil, nil, nil,
		))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetAuthUser(r.Context())
		if user == nil || user.LegacySessionID != "current-legacy-session" {
			t.Fatalf("legacy session id = %q", user.LegacySessionID)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: jwtCookieName, Value: token})
	request.AddCookie(&http.Cookie{Name: "DDusrSession_id", Value: "current-legacy-session"})
	recorder := httptest.NewRecorder()

	AuthMiddleware(svc)(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthMiddlewareRevalidatesMobileBearerAndPreservesSessionID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	svc := service.NewAuthService(repo, nil, &config.Config{JWT: config.JWTConfig{
		Secret: "fixture-secret",
		MaxAge: time.Hour,
	}}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	token, err := svc.GenerateMobileJWT(&model.AuthUser{USRSeq: 42, USRName: "Stale Name", USRStatus: "CCC"}, "mobile-family")
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(principalRows().AddRow(
			42, "member", "Current Name", "current@example.com", "CCC", nil,
			"approved", nil, nil, nil, nil, nil, nil,
		))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(42, "mobile-family").
		WillReturnRows(sqlmock.NewRows([]string{"ACTIVE"}).AddRow(true))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetAuthUser(r.Context())
		if user == nil {
			t.Fatal("current principal was not attached")
		}
		if user.USRName != "Current Name" || user.Email != "current@example.com" {
			t.Fatalf("principal = %#v", user)
		}
		if user.SessionID != "mobile-family" {
			t.Fatalf("session id = %q", user.SessionID)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	AuthMiddleware(svc)(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthMiddlewareRejectsMobileBearerAfterSessionFamilyRevocation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	svc := service.NewAuthService(repo, nil, &config.Config{JWT: config.JWTConfig{
		Secret: "fixture-secret",
		MaxAge: time.Hour,
	}}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	token, err := svc.GenerateMobileJWT(&model.AuthUser{USRSeq: 42, USRStatus: "CCC"}, "revoked-family")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(principalRows().AddRow(
			42, "member", "Member", "member@example.com", "CCC", nil,
			"approved", nil, nil, nil, nil, nil, nil,
		))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(42, "revoked-family").
		WillReturnRows(sqlmock.NewRows([]string{"ACTIVE"}).AddRow(false))

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	AuthMiddleware(svc)(next).ServeHTTP(recorder, request)

	if called {
		t.Fatal("revoked mobile session reached authenticated handler")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthMiddlewareRejectsLegacyCookieWhenCurrentAccountBecomesIneligible(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	svc := service.NewAuthService(repo, nil, &config.Config{}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS")).
		WithArgs("legacy-session").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS"}).
			AddRow(42, "member", "Member", "CCC"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(principalRows().AddRow(
			42, "member", "Member", "member@example.com", "AAA", nil,
			"approved", nil, nil, nil, nil, nil, nil,
		))

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	request.AddCookie(&http.Cookie{Name: "DDusrSession_id", Value: "legacy-session"})
	recorder := httptest.NewRecorder()

	AuthMiddleware(svc)(next).ServeHTTP(recorder, request)

	if called {
		t.Fatal("ineligible current principal reached authenticated handler")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthMiddlewareAllowsLegacyPendingStatusThroughCentralPolicy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	svc := service.NewAuthService(repo, nil, &config.Config{}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS")).
		WithArgs("pending-legacy-session").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS"}).
			AddRow(42, "member", "Member", "BBB"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(principalRows().AddRow(
			42, "member", "Member", "member@example.com", "BBB", nil,
			"pending", nil, nil, nil, nil, time.Now(), nil,
		))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetAuthUser(r.Context())
		if user == nil || user.USRStatus != "BBB" || user.LegacySessionID != "pending-legacy-session" {
			t.Fatalf("principal = %#v", user)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	request.AddCookie(&http.Cookie{Name: "DDusrSession_id", Value: "pending-legacy-session"})
	recorder := httptest.NewRecorder()

	AuthMiddleware(svc)(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOptionalAuthMiddlewareDropsIneligibleCurrentPrincipal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	svc := service.NewAuthService(repo, nil, &config.Config{JWT: config.JWTConfig{
		Secret: "fixture-secret",
		MaxAge: time.Hour,
	}}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	token, err := svc.GenerateJWT(&model.AuthUser{USRSeq: 42, USRStatus: "CCC"})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(principalRows().AddRow(
			42, "member", "Member", "member@example.com", "AAA", nil,
			"approved", nil, nil, nil, nil, nil, nil,
		))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user := GetAuthUser(r.Context()); user != nil {
			t.Fatalf("optional auth retained ineligible principal: %#v", user)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/api/feed", nil)
	request.AddCookie(&http.Cookie{Name: jwtCookieName, Value: token})
	recorder := httptest.NewRecorder()

	OptionalAuthMiddleware(svc)(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthMiddlewarePreservesCompanionLegacySessionForMobileBearerLogout(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	svc := service.NewAuthService(repo, nil, &config.Config{JWT: config.JWTConfig{
		Secret: "fixture-secret",
		MaxAge: time.Hour,
	}}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	token, err := svc.GenerateMobileJWT(&model.AuthUser{USRSeq: 42, USRStatus: "CCC"}, "mobile-family")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(principalRows().AddRow(
			42, "member", "Member", "member@example.com", "CCC", nil,
			"approved", nil, nil, nil, nil, nil, nil,
		))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(42, "mobile-family").
		WillReturnRows(sqlmock.NewRows([]string{"active"}).AddRow(true))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetAuthUser(r.Context())
		if user == nil || user.SessionID != "mobile-family" || user.LegacySessionID != "current-legacy-session" {
			t.Fatalf("session metadata = %#v", user)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.AddCookie(&http.Cookie{Name: "DDusrSession_id", Value: "current-legacy-session"})
	recorder := httptest.NewRecorder()

	AuthMiddleware(svc)(next).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func principalRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
		"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
	})
}
