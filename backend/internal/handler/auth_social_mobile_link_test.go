package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

func TestSocialLinkAcceptsCanonicalMobileRequest(t *testing.T) {
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
	digest := sha256.Sum256([]byte("fixture-link-token"))
	tokenHash := hex.EncodeToString(digest[:])

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT SLC_PROVIDER, SLC_SUBJECT, SLC_EMAIL, SLC_STATUS, SLC_EXPIRES_AT`).
		WithArgs(tokenHash).
		WillReturnRows(sqlmock.NewRows([]string{
			"SLC_PROVIDER", "SLC_SUBJECT", "SLC_EMAIL", "SLC_STATUS", "SLC_EXPIRES_AT",
		}).AddRow("KT", "fixture-provider-subject", "provider@example.com", "READY", time.Now().Add(time.Minute)))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD[\s\S]+ON DUPLICATE KEY UPDATE`).
		WithArgs("KT", "fixture-provider-subject", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT SLR_FAILED_ATTEMPTS, SLR_LOCKED_AT[\s\S]+FOR UPDATE`).
		WithArgs("KT", "fixture-provider-subject").
		WillReturnRows(sqlmock.NewRows([]string{"SLR_FAILED_ATTEMPTS", "SLR_LOCKED_AT"}).AddRow(0, nil))
	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\)`).
		WithArgs("member@example.com", service.MysqlNativePassword("fixture-password")).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", nil, "10", "member@example.com", nil, nil))
	mock.ExpectQuery(`SELECT USR_SEQ\s+FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "fixture-provider-subject").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}))
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT", "fixture-provider-subject", "provider@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_LINK_CONTINUATION`).
		WithArgs(tokenHash).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "CCC", nil,
			"approved", nil, "10", "International", nil, time.Now(), time.Now()))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 42, "CCC").
		WillReturnResult(sqlmock.NewResult(1, 1))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/social/link",
		strings.NewReader(`{"linkToken":"fixture-link-token","email":"member@example.com","password":"fixture-password"}`),
	)
	recorder := httptest.NewRecorder()
	handler.SocialLink(recorder, request)

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
		t.Fatal(err)
	}
}

func TestSocialLinkWebCompletesCanonicalRequestWithCookieBridge(t *testing.T) {
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
	digest := sha256.Sum256([]byte("fixture-link-token"))
	tokenHash := hex.EncodeToString(digest[:])

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT SLC_PROVIDER, SLC_SUBJECT, SLC_EMAIL, SLC_STATUS, SLC_EXPIRES_AT`).
		WithArgs(tokenHash).
		WillReturnRows(sqlmock.NewRows([]string{
			"SLC_PROVIDER", "SLC_SUBJECT", "SLC_EMAIL", "SLC_STATUS", "SLC_EXPIRES_AT",
		}).AddRow("KT", "fixture-provider-subject", "provider@example.com", "READY", time.Now().Add(time.Minute)))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD[\s\S]+ON DUPLICATE KEY UPDATE`).
		WithArgs("KT", "fixture-provider-subject", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT SLR_FAILED_ATTEMPTS, SLR_LOCKED_AT[\s\S]+FOR UPDATE`).
		WithArgs("KT", "fixture-provider-subject").
		WillReturnRows(sqlmock.NewRows([]string{"SLR_FAILED_ATTEMPTS", "SLR_LOCKED_AT"}).AddRow(0, nil))
	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\)`).
		WithArgs("member@example.com", service.MysqlNativePassword("fixture-password")).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", nil, "10", "member@example.com", nil, nil))
	mock.ExpectQuery(`SELECT USR_SEQ\s+FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "fixture-provider-subject").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}))
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT", "fixture-provider-subject", "provider@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_LINK_CONTINUATION`).
		WithArgs(tokenHash).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "CCC", nil,
			"approved", nil, "10", "International", nil, time.Now(), time.Now()))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE WEO_MEMBER")).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO WEO_MEMBER_LOG")).
		WithArgs(42, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/social/link/web",
		strings.NewReader(`{"linkToken":"fixture-link-token","email":"member@example.com","password":"fixture-password"}`),
	)
	recorder := httptest.NewRecorder()
	handler.SocialLinkWeb(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(recorder.Result().Cookies()) != 6 {
		t.Fatalf("cookie count = %d", len(recorder.Result().Cookies()))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSocialLinkRejectsLegacyPhoneMergePayload(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	cacheStore := cache.New(time.Minute, time.Minute)
	svc := service.NewAuthService(repo, nil, cfg, cacheStore, nil, zerolog.Nop())
	handler := NewAuthHandler(svc, service.NewMemberService(repo), nil, nil, cacheStore, cfg, zerolog.Nop())

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/social/link",
		strings.NewReader(`{"token":"legacy-token","mode":"merge","phone":"01000000000"}`),
	)
	recorder := httptest.NewRecorder()
	handler.SocialLink(recorder, request)

	if recorder.Code != http.StatusGone {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRespondSocialLinkErrorMapsLockedContinuationToRateLimit(t *testing.T) {
	recorder := httptest.NewRecorder()
	respondSocialLinkError(recorder, repository.ErrSocialLinkReauthLocked)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"REAUTHENTICATION_LOCKED"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}
