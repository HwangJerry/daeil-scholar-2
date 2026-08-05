package service

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/social-auth/kakao"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type fakeKakaoAuthClient struct {
	result kakao.AuthResult
	err    error
}

func (f fakeKakaoAuthClient) AuthenticateByCode(context.Context, string, string) (kakao.AuthResult, error) {
	return f.result, f.err
}

func (f fakeKakaoAuthClient) AuthenticateByAccessToken(context.Context, string) (kakao.AuthResult, error) {
	return f.result, f.err
}

func (f fakeKakaoAuthClient) Logout(context.Context, string) error {
	return f.err
}

func TestAuthenticateKakaoMobileReturnsCanonicalAuthenticatedSessionForApprovedLinkedMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	provider := fakeKakaoAuthClient{result: kakao.AuthResult{Profile: kakao.UserProfile{
		KakaoID:  "fixture-provider-id",
		Email:    "provider@example.com",
		Nickname: "Provider Member",
	}}}
	svc := NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), provider, zerolog.Nop())

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

	result, err := svc.AuthenticateKakaoMobile(context.Background(), "fixture-provider-token")
	if err != nil {
		t.Fatalf("AuthenticateKakaoMobile: %v", err)
	}
	if result.Status != model.SocialAuthAuthenticated || result.Session == nil {
		t.Fatalf("result = %#v", result)
	}
	if result.Session.User.Verification.Status != model.VerificationApproved {
		t.Fatalf("verification = %#v", result.Session.User.Verification)
	}
	if result.Session.User.USRStatus != "" {
		t.Fatalf("legacy usrStatus must not be exposed in canonical session: %#v", result.Session.User)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestAuthenticateKakaoMobileDoesNotAutoLinkMatchingProviderEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cacheStore := cache.New(time.Minute, time.Minute)
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	provider := fakeKakaoAuthClient{result: kakao.AuthResult{Profile: kakao.UserProfile{
		KakaoID:  "fixture-unlinked-provider-id",
		Email:    "existing-member@example.com",
		Nickname: "Provider Member",
	}}}
	svc := NewAuthService(repo, nil, cfg, cacheStore, provider, zerolog.Nop())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS")).
		WithArgs("KT", "fixture-unlinked-provider-id").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD")).
		WithArgs("KT", "fixture-unlinked-provider-id", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_SOCIAL_LINK_CONTINUATION")).
		WithArgs(sqlmock.AnyArg(), "KT", "fixture-unlinked-provider-id", "existing-member@example.com", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := svc.AuthenticateKakaoMobile(context.Background(), "fixture-provider-token")
	if err != nil {
		t.Fatalf("AuthenticateKakaoMobile: %v", err)
	}
	if result.Status != model.SocialAuthLinkRequired || result.LinkRequired == nil {
		t.Fatalf("result = %#v", result)
	}
	if result.LinkRequired.Provider != model.SocialProviderKakao {
		t.Fatalf("provider = %q", result.LinkRequired.Provider)
	}
	if result.LinkRequired.Profile.Email != "existing-member@example.com" {
		t.Fatalf("profile = %#v", result.LinkRequired.Profile)
	}
	if result.LinkRequired.LinkToken == "" || result.LinkRequired.ExpiresAt <= time.Now().Unix() {
		t.Fatalf("invalid continuation metadata: %#v", result.LinkRequired)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQL lookup indicates auto-link attempt: %v", err)
	}
}

func TestAuthenticateKakaoMobileIssuesRestrictedSessionForPendingLinkedMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	provider := fakeKakaoAuthClient{result: kakao.AuthResult{Profile: kakao.UserProfile{
		KakaoID: "fixture-pending-provider-id",
	}}}
	svc := NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), provider, zerolog.Nop())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS")).
		WithArgs("KT", "fixture-pending-provider-id").
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(43, "pending-member", "Pending Member", "BBB", "01000000001", "11", "pending@example.com", nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(43).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(43, "pending-member", "Pending Member", "pending@example.com", "BBB", nil,
			"pending", nil, "11", "International", nil, time.Now(), nil))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 43, "BBB").
		WillReturnResult(sqlmock.NewResult(1, 1))

	result, err := svc.AuthenticateKakaoMobile(context.Background(), "fixture-provider-token")
	if err != nil {
		t.Fatalf("AuthenticateKakaoMobile: %v", err)
	}
	if result.Status != model.SocialAuthAuthenticated || result.Session == nil {
		t.Fatalf("result = %#v", result)
	}
	if result.Session.User.Verification.Status != model.VerificationPending {
		t.Fatalf("verification status = %q", result.Session.User.Verification.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestIssueMobileSessionRejectsFreshlyIneligiblePrincipal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	svc := NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "AAA", nil,
			"approved", nil, nil, nil, nil, nil, nil))

	_, err = svc.IssueMobileSession(&model.User{USRSeq: 42, USRStatus: "CCC"})
	if !errors.Is(err, ErrLoginWithdrawn) {
		t.Fatalf("IssueMobileSession error = %v, want ErrLoginWithdrawn", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueMobileSessionRejectsStatusChangeAtRefreshPersistence(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	svc := NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	expectCanonicalPrincipal(mock, 42, "CCC", "approved")
	mock.ExpectExec(`SELECT \?, m.USR_SEQ, \?, \?, NOW\(\)`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 42, "CCC").
		WillReturnResult(sqlmock.NewResult(0, 0))

	session, err := svc.IssueMobileSession(&model.User{USRSeq: 42, USRStatus: "CCC"})
	if !errors.Is(err, repository.ErrSessionPrincipalChanged) {
		t.Fatalf("IssueMobileSession error = %v", err)
	}
	if session != nil {
		t.Fatalf("session returned after status changed at persistence: %#v", session)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteCanonicalSocialLinkRecoversThroughLinkedLoginAfterSessionIssueFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "fixture-secret", MaxAge: time.Hour}}
	provider := fakeKakaoAuthClient{result: kakao.AuthResult{Profile: kakao.UserProfile{
		KakaoID: "fixture-provider-id",
	}}}
	svc := NewAuthService(repo, nil, cfg, cache.New(time.Minute, time.Minute), provider, zerolog.Nop())
	tokenHash := hashSocialLinkToken("fixture-link-token")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT SLC_PROVIDER, SLC_SUBJECT, SLC_EMAIL, SLC_STATUS, SLC_EXPIRES_AT`).
		WithArgs(tokenHash).
		WillReturnRows(sqlmock.NewRows([]string{
			"SLC_PROVIDER", "SLC_SUBJECT", "SLC_EMAIL", "SLC_STATUS", "SLC_EXPIRES_AT",
		}).AddRow("KT", "fixture-provider-id", "provider@example.com", "READY", time.Now().Add(time.Minute)))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD[\s\S]+ON DUPLICATE KEY UPDATE`).
		WithArgs("KT", "fixture-provider-id", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT SLR_FAILED_ATTEMPTS, SLR_LOCKED_AT[\s\S]+FOR UPDATE`).
		WithArgs("KT", "fixture-provider-id").
		WillReturnRows(sqlmock.NewRows([]string{"SLR_FAILED_ATTEMPTS", "SLR_LOCKED_AT"}).AddRow(0, nil))
	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\)`).
		WithArgs("member@example.com", MysqlNativePassword("fixture-password")).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", nil, "10", "member@example.com", nil, nil))
	mock.ExpectQuery(`SELECT USR_SEQ\s+FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "fixture-provider-id").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}))
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT", "fixture-provider-id", "provider@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_LINK_CONTINUATION`).
		WithArgs(tokenHash).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectCanonicalPrincipal(mock, 42, "CCC", "approved")
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 42, "CCC").
		WillReturnError(errors.New("fixture session insert failure"))

	result, err := svc.CompleteCanonicalSocialLink("fixture-link-token", "member@example.com", "fixture-password")
	if err == nil {
		t.Fatal("CompleteCanonicalSocialLink error = nil")
	}
	if result.Status != "" || result.Session != nil {
		t.Fatalf("session was returned after persistence failure: %#v", result)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS")).
		WithArgs("KT", "fixture-provider-id").
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", nil, "10", "member@example.com", nil, nil))
	expectCanonicalPrincipal(mock, 42, "CCC", "approved")
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 42, "CCC").
		WillReturnResult(sqlmock.NewResult(1, 1))

	retry, err := svc.AuthenticateKakaoMobile(context.Background(), "fixture-provider-token")
	if err != nil {
		t.Fatalf("AuthenticateKakaoMobile retry: %v", err)
	}
	if retry.Status != model.SocialAuthAuthenticated || retry.Session == nil {
		t.Fatalf("retry result = %#v", retry)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectCanonicalPrincipal(mock sqlmock.Sqlmock, usrSeq int, status string, verification string) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(usrSeq).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(usrSeq, "member", "Member", "member@example.com", status, nil,
			verification, nil, "10", "International", nil, time.Now(), time.Now()))
}
