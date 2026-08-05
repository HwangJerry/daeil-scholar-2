package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/dflh-saf/social-auth/kakao"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type handlerFakeKakaoClient struct {
	result kakao.AuthResult
	err    error
}

func (f handlerFakeKakaoClient) AuthenticateByCode(context.Context, string, string) (kakao.AuthResult, error) {
	return f.result, f.err
}

func (f handlerFakeKakaoClient) AuthenticateByAccessToken(context.Context, string) (kakao.AuthResult, error) {
	return f.result, f.err
}

func (f handlerFakeKakaoClient) Logout(context.Context, string) error {
	return f.err
}

func TestKakaoMobileLoginUsesCanonicalRequestAndAuthenticatedEnvelope(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	cacheStore := cache.New(time.Minute, time.Minute)
	provider := handlerFakeKakaoClient{result: kakao.AuthResult{Profile: kakao.UserProfile{
		KakaoID:  "fixture-provider-id",
		Email:    "provider@example.com",
		Nickname: "Provider Member",
	}}}
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, provider, zerolog.Nop())
	handler := NewAuthHandler(svc, nil, nil, nil, cacheStore, cfg, zerolog.Nop())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS")).
		WithArgs("KT", "fixture-provider-id").
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", "01000000000", "10", "member@example.com", nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "CCC", nil,
			"approved", nil, "10", "International", nil, nil, nil))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 42, "CCC").
		WillReturnResult(sqlmock.NewResult(1, 1))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/kakao/mobile",
		strings.NewReader(`{"grantType":"access_token","accessToken":"fixture-provider-token"}`),
	)
	recorder := httptest.NewRecorder()
	handler.KakaoMobileLogin(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["status"] != "authenticated" || response["session"] == nil {
		t.Fatalf("canonical response = %#v", response)
	}
	for _, legacyField := range []string{"accessToken", "refreshToken", "usrSeq", "usrId", "usrName", "usrStatus", "sid", "jti"} {
		if _, exists := response[legacyField]; exists {
			t.Fatalf("legacy flat field %q must be absent: %#v", legacyField, response)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestKakaoMobileLoginMapsLinkedWithdrawnAccountToForbidden(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	cacheStore := cache.New(time.Minute, time.Minute)
	provider := handlerFakeKakaoClient{result: kakao.AuthResult{Profile: kakao.UserProfile{KakaoID: "fixture-provider-id"}}}
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, provider, zerolog.Nop())
	handler := NewAuthHandler(svc, nil, nil, nil, cacheStore, cfg, zerolog.Nop())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS")).
		WithArgs("KT", "fixture-provider-id").
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "AAA", nil, "10", "member@example.com", nil, nil))

	request := httptest.NewRequest(http.MethodPost, "/api/auth/kakao/mobile",
		strings.NewReader(`{"grantType":"access_token","accessToken":"fixture-provider-token"}`))
	recorder := httptest.NewRecorder()
	handler.KakaoMobileLogin(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"ACCOUNT_WITHDRAWN"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestKakaoMobileLoginRejectsAuthorizationCodeGrant(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{
		JWT:   config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour},
		Kakao: config.KakaoConfig{RedirectURI: "https://example.invalid/api/auth/kakao/callback"},
	}
	cacheStore := cache.New(time.Minute, time.Minute)
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, handlerFakeKakaoClient{}, zerolog.Nop())
	handler := NewAuthHandler(svc, nil, nil, nil, cacheStore, cfg, zerolog.Nop())
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/kakao/mobile",
		strings.NewReader(`{"grantType":"authorization_code","code":"fixture-code","redirectUri":"https://example.invalid/api/auth/kakao/callback"}`),
	)
	recorder := httptest.NewRecorder()

	handler.KakaoMobileLogin(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestMeReloadsCanonicalPrincipalFromDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	cacheStore := cache.New(time.Minute, time.Minute)
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, handlerFakeKakaoClient{}, zerolog.Nop())
	handler := NewAuthHandler(svc, nil, nil, nil, cacheStore, cfg, zerolog.Nop())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "CCC", nil,
			"approved", nil, "10", "International", nil, nil, nil))

	request := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42}))
	recorder := httptest.NewRecorder()
	handler.Me(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["email"] != "member@example.com" {
		t.Fatalf("email = %#v", response["email"])
	}
	verification, ok := response["verification"].(map[string]any)
	if !ok || verification["status"] != "approved" {
		t.Fatalf("verification = %#v", response["verification"])
	}
	if _, exists := response["usrStatus"]; exists {
		t.Fatalf("legacy usrStatus must not be exposed: %#v", response)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMobileLoginUsesCanonicalEmailAndAuthenticatedEnvelope(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	cacheStore := cache.New(time.Minute, time.Minute)
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, handlerFakeKakaoClient{}, zerolog.Nop())
	handler := NewAuthHandler(svc, service.NewMemberService(repo), nil, nil, cacheStore, cfg, zerolog.Nop())

	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\) AND USR_PWD = \?`).
		WithArgs("member@example.com", service.MysqlNativePassword("fixture-password")).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(43, "legacy-id", "Pending Member", "BBB", nil, "11", "member@example.com", nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(43).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(43, "legacy-id", "Pending Member", "member@example.com", "BBB", nil,
			"pending", nil, "11", "International", nil, time.Now(), nil))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 43, "BBB").
		WillReturnResult(sqlmock.NewResult(1, 1))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/mobile/login",
		strings.NewReader(`{"email":"member@example.com","password":"fixture-password"}`),
	)
	recorder := httptest.NewRecorder()
	handler.MobileLogin(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["status"] != "authenticated" || response["session"] == nil {
		t.Fatalf("canonical response = %#v", response)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMobileLoginRejectsFreshlyIneligiblePrincipal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	cacheStore := cache.New(time.Minute, time.Minute)
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, handlerFakeKakaoClient{}, zerolog.Nop())
	handler := NewAuthHandler(svc, service.NewMemberService(repo), nil, nil, cacheStore, cfg, zerolog.Nop())

	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\) AND USR_PWD = \?`).
		WithArgs("member@example.com", service.MysqlNativePassword("fixture-password")).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(43, "legacy-id", "Member", "CCC", nil, "11", "member@example.com", nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(43).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(43, "legacy-id", "Member", "member@example.com", "AAA", nil,
			"approved", nil, "11", "International", nil, time.Now(), time.Now()))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/mobile/login",
		strings.NewReader(`{"email":"member@example.com","password":"fixture-password"}`),
	)
	recorder := httptest.NewRecorder()
	handler.MobileLogin(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var response model.APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "ACCOUNT_WITHDRAWN" {
		t.Fatalf("code = %q", response.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestLoginRejectsFreshlyIneligiblePrincipal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	cacheStore := cache.New(time.Minute, time.Minute)
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, handlerFakeKakaoClient{}, zerolog.Nop())
	handler := NewAuthHandler(svc, service.NewMemberService(repo), nil, nil, cacheStore, cfg, zerolog.Nop())

	mock.ExpectQuery(`WHERE USR_ID = \? AND USR_PWD = \? AND USR_STATUS IN \('CCC', 'ZZZ'\)`).
		WithArgs("member", service.MysqlNativePassword("fixture-password")).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", nil, "10", "member@example.com", nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "AAA", nil,
			"approved", nil, "10", "International", nil, time.Now(), time.Now()))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/login",
		strings.NewReader(`{"usrId":"member","password":"fixture-password"}`),
	)
	recorder := httptest.NewRecorder()
	handler.Login(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var response model.APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "ACCOUNT_WITHDRAWN" {
		t.Fatalf("code = %q", response.Code)
	}
	if cookies := recorder.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("issued %d auth cookies for ineligible principal", len(cookies))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMeRejectsFreshlyIneligiblePrincipal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
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

	request := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42, USRStatus: "CCC"}))
	recorder := httptest.NewRecorder()
	handler.Me(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
