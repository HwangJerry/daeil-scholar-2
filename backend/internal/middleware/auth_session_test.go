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

func TestRevokedMobileSessionRejectsExistingAccessTokenImmediately(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
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

	expectCurrentMember(mock)
	mock.ExpectExec(`INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(sqlmock.AnyArg(), 42, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	session, err := service.NewMobileSessionIssuer(auth).Issue(&model.User{
		USRSeq:    42,
		USRID:     "member",
		USRName:   "Member",
		USRStatus: "CCC",
	})
	if err != nil {
		t.Fatal(err)
	}

	protected := AuthMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	expectCurrentMember(mock)
	mock.ExpectQuery(`FROM ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(42, session.SID).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
	first := authenticatedRequest(session.AccessToken)
	firstRecorder := httptest.NewRecorder()
	protected.ServeHTTP(firstRecorder, first)
	if firstRecorder.Code != http.StatusNoContent {
		t.Fatalf("active session status = %d", firstRecorder.Code)
	}

	mock.ExpectExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(42, session.SID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := authRepo.RevokeMobileSession(42, session.SID); err != nil {
		t.Fatal(err)
	}

	expectCurrentMember(mock)
	mock.ExpectQuery(`FROM ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(42, session.SID).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))
	second := authenticatedRequest(session.AccessToken)
	secondRecorder := httptest.NewRecorder()
	protected.ServeHTTP(secondRecorder, second)
	if secondRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status = %d, want 401", secondRecorder.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func authenticatedRequest(accessToken string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	return request
}

func expectCurrentMember(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`LEFT JOIN ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "EMAIL", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT",
			"REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "CCC", "member@example.com", nil, "approved", nil, nil, nil, nil, nil, nil))
}
