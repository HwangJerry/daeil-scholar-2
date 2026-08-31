package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

func TestSocialLinkValidationFailureDoesNotConsumeToken(t *testing.T) {
	tokenStore := service.NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	if _, err := tokenStore.Put("retry-token", model.SocialLinkData{}, time.Minute); err != nil {
		t.Fatal(err)
	}
	handler := &AuthHandler{socialLinkTokens: tokenStore}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/social/link",
		strings.NewReader(`{
			"token":"retry-token",
			"mode":"new",
			"name":"Member",
			"email":"relay@privaterelay.appleid.com",
			"phone":"010-1234-5678",
			"fn":"not-a-number",
			"fmDept":"영어"
		}`),
	)
	response := httptest.NewRecorder()

	handler.SocialLink(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if _, err := tokenStore.Begin("retry-token"); err != nil {
		t.Fatalf("validation failure consumed token: %v", err)
	}
}

func TestSocialLinkRejectsUnsupportedModeWithoutConsumingToken(t *testing.T) {
	tokenStore := service.NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	if _, err := tokenStore.Put("unsupported-mode-token", model.SocialLinkData{}, time.Minute); err != nil {
		t.Fatal(err)
	}
	handler := &AuthHandler{socialLinkTokens: tokenStore}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/social/link",
		strings.NewReader(`{"token":"unsupported-mode-token","mode":"unsupported"}`),
	)
	response := httptest.NewRecorder()

	handler.SocialLink(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"UNSUPPORTED_SOCIAL_LINK_MODE"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
	if _, err := tokenStore.Begin("unsupported-mode-token"); err != nil {
		t.Fatalf("unsupported mode consumed token: %v", err)
	}
}

func TestSocialLinkReturnsConflictForAlreadyLinkedSocialAccount(t *testing.T) {
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
	tokenStore := service.NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	if _, err := tokenStore.Put("duplicate-token", model.SocialLinkData{
		Provider: "KT",
		SocialID: "subject",
		Email:    "member@example.com",
	}, time.Minute); err != nil {
		t.Fatal(err)
	}
	handler := NewAuthHandler(
		auth,
		service.NewMemberService(authRepo),
		nil,
		cache.New(time.Minute, time.Minute),
		tokenStore,
		cfg,
		zerolog.Nop(),
	)

	mock.ExpectQuery(`WHERE \(USR_PHONE = \? OR`).
		WithArgs("01012345678", "01012345678").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate provider subject"})
	mock.ExpectRollback()

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/social/link",
		strings.NewReader(`{
			"token":"duplicate-token",
			"mode":"new",
			"name":"Member",
			"email":"member@example.com",
			"phone":"010-1234-5678",
			"fn":"31",
			"fmDept":"영어"
		}`),
	)
	response := httptest.NewRecorder()

	handler.SocialLink(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"SOCIAL_ACCOUNT_ALREADY_LINKED"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSocialLinkRequiresOwnershipConfirmationForExistingPhoneRegardlessOfName(t *testing.T) {
	tests := []struct {
		name          string
		existingName  string
		requestedName string
	}{
		{
			name:          "different name",
			existingName:  "김동문",
			requestedName: "홍동문",
		},
		{
			name:          "matching name",
			existingName:  "홍 길동",
			requestedName: "홍길동",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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
			tokenStore := service.NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
			if _, err := tokenStore.Put("phone-conflict-token", model.SocialLinkData{
				Provider: "KT",
				SocialID: "subject",
				Email:    "member@example.com",
			}, time.Minute); err != nil {
				t.Fatal(err)
			}
			handler := NewAuthHandler(
				auth,
				service.NewMemberService(authRepo),
				nil,
				cache.New(time.Minute, time.Minute),
				tokenStore,
				cfg,
				zerolog.Nop(),
			)

			mock.ExpectQuery(`WHERE \(USR_PHONE = \? OR`).
				WithArgs("01012345678", "01012345678").
				WillReturnRows(sqlmock.NewRows([]string{
					"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE",
					"USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
				}).AddRow(42, "existing-member", test.existingName, "CCC", "01012345678", "31", "existing@example.com", nil, nil))

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/auth/social/link",
				strings.NewReader(`{
					"token":"phone-conflict-token",
					"mode":"new",
					"name":"`+test.requestedName+`",
					"email":"member@example.com",
					"phone":"010-1234-5678",
					"fn":"31",
					"fmDept":"영어"
				}`),
			)
			response := httptest.NewRecorder()

			handler.SocialLink(response, request)

			if response.Code != http.StatusConflict {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), `"code":"OWNERSHIP_CONFIRMATION_REQUIRED"`) {
				t.Fatalf("body = %s", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
