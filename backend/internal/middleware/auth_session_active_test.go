// auth_session_active_test.go — proves logout immediately invalidates a
// still-unexpired mobile access token via the per-request session-active check.
package middleware

import (
	"net/http"
	"net/http/httptest"
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

func newAuthServiceForMiddlewareTest(t *testing.T) (*service.AuthService, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret",
			MaxAge: 24 * time.Hour,
		},
	}
	authRepo := repository.NewAuthRepository(sqlxDB)
	authService := service.NewAuthService(
		authRepo,
		repository.NewSessionRepository(sqlxDB),
		cfg,
		cache.New(time.Minute, time.Minute),
		nil,
		zerolog.Nop(),
	)
	return authService, mock, func() { _ = db.Close() }
}

func callWithBearerToken(authService *service.AuthService, token string) *httptest.ResponseRecorder {
	handler := AuthMiddleware(authService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestMobileAccessTokenAcceptedWhileSessionActive(t *testing.T) {
	authService, mock, closeDB := newAuthServiceForMiddlewareTest(t)
	defer closeDB()

	const usrSeq = 42
	const sid = "session-abc"
	token, err := authService.GenerateMobileJWT(&model.AuthUser{
		USRSeq:    usrSeq,
		USRName:   "Member",
		USRStatus: "CCC",
	}, sid)
	if err != nil {
		t.Fatalf("GenerateMobileJWT: %v", err)
	}

	mock.ExpectQuery(`SELECT COUNT\(\*\)\s+FROM ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(usrSeq, sid).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rec := callWithBearerToken(authService, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 while session active, got %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMobileAccessTokenRejectedAfterSessionRevoked(t *testing.T) {
	authService, mock, closeDB := newAuthServiceForMiddlewareTest(t)
	defer closeDB()

	const usrSeq = 42
	const sid = "session-abc"
	// The token itself is still validly signed and unexpired - only the
	// session behind it has been revoked (e.g. the user logged out).
	token, err := authService.GenerateMobileJWT(&model.AuthUser{
		USRSeq:    usrSeq,
		USRName:   "Member",
		USRStatus: "CCC",
	}, sid)
	if err != nil {
		t.Fatalf("GenerateMobileJWT: %v", err)
	}

	// Simulate RevokeMobileRefreshTokensBySession having run: no active,
	// non-expired, non-revoked rows remain for this session.
	mock.ExpectQuery(`SELECT COUNT\(\*\)\s+FROM ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(usrSeq, sid).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	rec := callWithBearerToken(authService, token)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after session revoked, got %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
