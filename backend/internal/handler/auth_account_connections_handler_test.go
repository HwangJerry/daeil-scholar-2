package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type stubIdentityLinkVerifier struct {
	account service.VerifiedSocialAccount
	err     error
}

func (s stubIdentityLinkVerifier) Provider() model.SocialProvider {
	return s.account.Identity.Provider
}

func (s stubIdentityLinkVerifier) Verify(context.Context, model.SocialAuthorization) (service.VerifiedSocialAccount, error) {
	return s.account, s.err
}

func TestLinkIdentityHandlerConnectsKakaoAndApple(t *testing.T) {
	tests := []struct {
		name         string
		pathProvider string
		provider     model.SocialProvider
		body         string
	}{
		{name: "kakao", pathProvider: "kakao", provider: model.SocialProviderKakao, body: `{"accessToken":"kakao-token"}`},
		{name: "apple", pathProvider: "apple", provider: model.SocialProviderApple, body: `{"challengeId":"challenge","identityToken":"identity-token","authorizationCode":"code","givenName":"길동","familyName":"홍"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, mock, cleanup := newIdentityLinkAuthHandlerForTest(t, stubIdentityLinkVerifier{
				account: service.VerifiedSocialAccount{Identity: model.VerifiedSocialIdentity{
					Provider: test.provider,
					Subject:  "provider-subject",
					Email:    "member@example.com",
				}},
			})
			defer cleanup()
			mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
				WithArgs(string(test.provider), "provider-subject").
				WillReturnError(sql.ErrNoRows)
			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
				WithArgs(42, string(test.provider), "provider-subject", "member@example.com").
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			expectHandlerAccountConnections(mock, []string{string(test.provider)}, true)

			recorder := serveLinkIdentityRequest(handler, test.pathProvider, test.body)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			var connections model.AccountConnections
			if err := json.Unmarshal(recorder.Body.Bytes(), &connections); err != nil {
				t.Fatal(err)
			}
			if len(connections.Providers) != 1 || connections.Providers[0] != string(test.provider) || !connections.HasPassword {
				t.Fatalf("connections = %#v", connections)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLinkIdentityHandlerRejectsIdentityLinkedElsewhere(t *testing.T) {
	handler, mock, cleanup := newIdentityLinkAuthHandlerForTest(t, stubIdentityLinkVerifier{
		account: service.VerifiedSocialAccount{Identity: model.VerifiedSocialIdentity{
			Provider: model.SocialProviderKakao,
			Subject:  "provider-subject",
		}},
	})
	defer cleanup()
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "provider-subject").
		WillReturnRows(handlerMemberRows(99))

	recorder := serveLinkIdentityRequest(handler, "kakao", `{"accessToken":"kakao-token"}`)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "SOCIAL_ACCOUNT_ALREADY_LINKED")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLinkIdentityHandlerRejectsInvalidProvider(t *testing.T) {
	handler, mock, cleanup := newAccountConnectionsAuthHandlerForTest(t)
	defer cleanup()

	recorder := serveLinkIdentityRequest(handler, "naver", `{}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "INVALID_PROVIDER")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLinkIdentityHandlerRejectsVerificationFailure(t *testing.T) {
	handler, mock, cleanup := newIdentityLinkAuthHandlerForTest(t, stubIdentityLinkVerifier{
		account: service.VerifiedSocialAccount{Identity: model.VerifiedSocialIdentity{Provider: model.SocialProviderApple}},
		err:     errors.New("invalid apple token"),
	})
	defer cleanup()

	recorder := serveLinkIdentityRequest(handler, "apple", `{"challengeId":"challenge","identityToken":"identity-token","authorizationCode":"code"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "SOCIAL_VERIFICATION_FAILED")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetAccountConnectionsHandler(t *testing.T) {
	handler, mock, cleanup := newAccountConnectionsAuthHandlerForTest(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT NMS_GATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}).AddRow("AP").AddRow("KT"))
	mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(1))

	request := authenticatedAccountRequest(http.MethodGet, "/api/auth/account/connections")
	recorder := httptest.NewRecorder()
	handler.GetAccountConnections(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var connections model.AccountConnections
	if err := json.Unmarshal(recorder.Body.Bytes(), &connections); err != nil {
		t.Fatal(err)
	}
	if !connections.HasPassword || len(connections.Providers) != 2 {
		t.Fatalf("connections = %#v", connections)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectSocialHandlerRejectsInvalidProvider(t *testing.T) {
	handler, mock, cleanup := newAccountConnectionsAuthHandlerForTest(t)
	defer cleanup()

	router := chi.NewRouter()
	router.Delete("/api/auth/social/{provider}", handler.DisconnectSocial)
	request := authenticatedAccountRequest(http.MethodDelete, "/api/auth/social/naver")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "INVALID_PROVIDER")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectSocialHandlerReturnsDisconnected(t *testing.T) {
	handler, mock, cleanup := newAccountConnectionsAuthHandlerForTest(t)
	defer cleanup()
	expectHandlerAccountConnections(mock, []string{"KT"}, true)
	mock.ExpectBegin()
	expectHandlerLockedLoginMethods(mock, []string{"KT"}, true)
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	expectHandlerAccountConnections(mock, nil, true)

	recorder := serveDisconnectRequest(handler, "/api/auth/social/kakao")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.SocialDisconnectResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialDisconnectStatusDisconnected || result.Connections.Providers == nil || len(result.Connections.Providers) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectSocialHandlerReturnsNotConnected(t *testing.T) {
	handler, mock, cleanup := newAccountConnectionsAuthHandlerForTest(t)
	defer cleanup()
	expectHandlerAccountConnections(mock, []string{"AP"}, true)

	recorder := serveDisconnectRequest(handler, "/api/auth/social/kakao")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.SocialDisconnectResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialDisconnectStatusNotConnected || len(result.Connections.Providers) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectSocialHandlerRejectsLastLoginMethod(t *testing.T) {
	handler, mock, cleanup := newAccountConnectionsAuthHandlerForTest(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT NMS_GATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}).AddRow("KT"))
	mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(0))
	mock.ExpectBegin()
	expectHandlerLockedLoginMethods(mock, []string{"KT"}, false)
	mock.ExpectRollback()

	recorder := serveDisconnectRequest(handler, "/api/auth/social/kakao")

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "LAST_LOGIN_METHOD")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectHandlerLockedLoginMethods(mock sqlmock.Sqlmock, providers []string, hasPassword bool) {
	rows := sqlmock.NewRows([]string{"NMS_GATE"})
	for _, provider := range providers {
		rows.AddRow(provider)
	}
	mock.ExpectQuery(`SELECT NMS_GATE[\s\S]*NMS_STATUS IN \('Y', 'ACTIVE'\)[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(rows)
	passwordPresent := 0
	if hasPassword {
		passwordPresent = 1
	}
	mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(passwordPresent))
}

func newAccountConnectionsAuthHandlerForTest(t *testing.T) (*AuthHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	auth := service.NewAuthService(
		repository.NewAuthRepository(sqlxDB),
		repository.NewSessionRepository(sqlxDB),
		&config.Config{},
		cache.New(time.Minute, time.Minute),
		zerolog.Nop(),
	)
	return &AuthHandler{service: auth, logger: zerolog.Nop()}, mock, func() { _ = db.Close() }
}

func newIdentityLinkAuthHandlerForTest(
	t *testing.T,
	verifier service.SocialIdentityVerifier,
) (*AuthHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	auth := service.NewAuthService(
		repository.NewAuthRepository(sqlxDB),
		repository.NewSessionRepository(sqlxDB),
		&config.Config{},
		cache.New(time.Minute, time.Minute),
		zerolog.Nop(),
	)
	return &AuthHandler{
		service:    auth,
		socialAuth: service.NewSocialAuthService(auth, nil, nil, nil, verifier),
		logger:     zerolog.Nop(),
	}, mock, func() { _ = db.Close() }
}

func authenticatedAccountRequest(method string, target string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	return request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42}))
}

func serveDisconnectRequest(handler *AuthHandler, target string) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	router.Delete("/api/auth/social/{provider}", handler.DisconnectSocial)
	request := authenticatedAccountRequest(http.MethodDelete, target)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func serveLinkIdentityRequest(handler *AuthHandler, provider string, body string) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	router.Post("/api/auth/identities/link/{provider}", handler.LinkIdentity)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/identities/link/"+provider, strings.NewReader(body))
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42}))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func handlerMemberRows(usrSeq int) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
	}).AddRow(usrSeq, "member", "Member", "CCC", "01012345678", "", "member@example.com", "", nil)
}

func expectHandlerAccountConnections(mock sqlmock.Sqlmock, providers []string, hasPassword bool) {
	rows := sqlmock.NewRows([]string{"NMS_GATE"})
	for _, provider := range providers {
		rows.AddRow(provider)
	}
	mock.ExpectQuery(`SELECT NMS_GATE`).WithArgs(42).WillReturnRows(rows)
	passwordPresent := 0
	if hasPassword {
		passwordPresent = 1
	}
	mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(passwordPresent))
}

func assertErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != want {
		t.Fatalf("error code = %q, want %q", body.Code, want)
	}
}
