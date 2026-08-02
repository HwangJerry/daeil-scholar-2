// auth_email_login_handler_test.go — Verifies canonical email login response behavior.
package handler

import (
	"bytes"
	"encoding/json"
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

func TestMobileLoginUsesEmailAndReturnsPendingSession(t *testing.T) {
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
	handler := &AuthHandler{
		memberSvc:    service.NewMemberService(authRepo),
		mobileIssuer: service.NewMobileSessionIssuer(auth),
		logger:       zerolog.Nop(),
	}
	password := "correct-password"
	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\) AND USR_PWD = \?`).
		WithArgs("member@example.com", service.MysqlNativePassword(password)).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE",
			"USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "legacy-id", "Pending Member", "BBB", nil, nil, "member@example.com", nil, nil))
	mock.ExpectQuery(`LEFT JOIN ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "EMAIL", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT",
			"REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(
			42, "legacy-id", "Pending Member", "BBB", "member@example.com", nil,
			"pending", nil, "18", "영어", nil, nil, nil,
		))
	mock.ExpectExec(`INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(sqlmock.AnyArg(), 42, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := bytes.NewBufferString(`{"email":"member@example.com","password":"correct-password"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/mobile/login", body)
	recorder := httptest.NewRecorder()
	handler.MobileLogin(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.SocialAuthResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialAuthAuthenticated || result.Session == nil {
		t.Fatalf("result = %#v", result)
	}
	if result.Session.User.Verification.Status != model.VerificationPending {
		t.Fatalf("verification = %#v", result.Session.User.Verification)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
