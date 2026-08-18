package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type stubSocialVerifier struct {
	account VerifiedSocialAccount
	err     error
}

func (s stubSocialVerifier) Provider() model.SocialProvider {
	return s.account.Identity.Provider
}

func (s stubSocialVerifier) Verify(context.Context, model.SocialAuthorization) (VerifiedSocialAccount, error) {
	return s.account, s.err
}

func TestSocialAuthDoesNotAutoLinkMatchingEmail(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()

	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "kakao-subject").
		WillReturnError(sql.ErrNoRows)

	verifier := stubSocialVerifier{account: VerifiedSocialAccount{
		Identity: model.VerifiedSocialIdentity{
			Provider:      model.SocialProviderKakao,
			Subject:       "kakao-subject",
			Email:         "existing@example.com",
			EmailVerified: true,
		},
		Profile: model.SocialProviderProfile{Email: "existing@example.com"},
	}}
	issuer := NewMobileSessionIssuer(auth)
	social := NewSocialAuthService(
		auth,
		issuer,
		NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute)),
		nil,
		verifier,
	)
	result, err := social.Authenticate(context.Background(), model.KakaoAuthorization{
		AccessToken: "opaque",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialAuthLinkRequired {
		t.Fatalf("status = %q, want linkRequired", result.Status)
	}
	if result.LinkRequired == nil || result.LinkRequired.Profile.Email != "existing@example.com" {
		t.Fatalf("email should be prefill only: %#v", result.LinkRequired)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSocialProvidersApplySameLoginEligibility(t *testing.T) {
	tests := []struct {
		name       string
		provider   model.SocialProvider
		status     string
		wantStatus model.SocialAuthStatus
		wantCode   string
	}{
		{name: "kakao pending", provider: model.SocialProviderKakao, status: "BBB", wantStatus: model.SocialAuthPending},
		{name: "apple pending", provider: model.SocialProviderApple, status: "BBB", wantStatus: model.SocialAuthPending},
		{name: "kakao withdrawn", provider: model.SocialProviderKakao, status: "AAA", wantStatus: model.SocialAuthRejected, wantCode: "ACCOUNT_WITHDRAWN"},
		{name: "apple suspended", provider: model.SocialProviderApple, status: "DDD", wantStatus: model.SocialAuthRejected, wantCode: "ACCOUNT_SUSPENDED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			auth, mock, cleanup := newAuthServiceForTest(t)
			defer cleanup()
			mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
				WithArgs(string(test.provider), "provider-subject").
				WillReturnRows(memberRows(test.status))
			verifier := stubSocialVerifier{account: VerifiedSocialAccount{
				Identity: model.VerifiedSocialIdentity{
					Provider: test.provider,
					Subject:  "provider-subject",
				},
			}}
			social := NewSocialAuthService(
				auth,
				NewMobileSessionIssuer(auth),
				NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute)),
				nil,
				verifier,
			)
			result, err := social.Authenticate(context.Background(), socialAuthorizationForTest(test.provider))
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != test.wantStatus {
				t.Fatalf("status = %q, want %q", result.Status, test.wantStatus)
			}
			if test.wantCode != "" && (result.Rejected == nil || result.Rejected.Code != test.wantCode) {
				t.Fatalf("rejection = %#v, want code %q", result.Rejected, test.wantCode)
			}
		})
	}
}

func socialAuthorizationForTest(provider model.SocialProvider) model.SocialAuthorization {
	if provider == model.SocialProviderApple {
		return model.AppleAuthorization{}
	}
	return model.KakaoAuthorization{}
}

func TestPasswordLoginAppliesSameEligibilityPolicy(t *testing.T) {
	tests := []struct {
		status  string
		wantErr error
	}{
		{status: "CCC"},
		{status: "ZZZ"},
		{status: "BBB", wantErr: ErrLoginPending},
		{status: "AAA", wantErr: ErrLoginWithdrawn},
		{status: "DDD", wantErr: ErrLoginSuspended},
	}
	for _, test := range tests {
		t.Run(test.status, func(t *testing.T) {
			auth, mock, cleanup := newAuthServiceForTest(t)
			defer cleanup()
			password := "correct-password"
			mock.ExpectQuery(`WHERE USR_ID = \? AND USR_PWD = \?`).
				WithArgs("member", MysqlNativePassword(password)).
				WillReturnRows(memberRows(test.status))
			memberService := NewMemberService(auth.repo)

			user, err := memberService.LoginWithPassword("member", password)

			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil && user == nil {
				t.Fatal("allowed member was not returned")
			}
			if test.wantErr != nil && user != nil {
				t.Fatal("forbidden member was returned")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMobileSessionIssuerUsesDistinctTTLs(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	mock.ExpectExec(`INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(sqlmock.AnyArg(), 42, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	issuer := NewMobileSessionIssuer(auth)
	now := time.Unix(1_800_000_000, 0)
	issuer.now = func() time.Time { return now }
	session, err := issuer.Issue(&model.User{
		USRSeq:    42,
		USRID:     "member",
		USRName:   "Member",
		USRStatus: "CCC",
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.AccessExpiresAt != now.Add(15*time.Minute).Unix() {
		t.Fatalf("access expiry = %d", session.AccessExpiresAt)
	}
	if session.RefreshExpiresAt != now.Add(30*24*time.Hour).Unix() {
		t.Fatalf("refresh expiry = %d", session.RefreshExpiresAt)
	}
	if session.AccessExpiresAt >= session.RefreshExpiresAt {
		t.Fatal("access token must expire before refresh token")
	}
}

func TestCompletedMobileSocialLinkReturnsServiceTokens(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	mock.ExpectExec(`INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(sqlmock.AnyArg(), 42, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	social := NewSocialAuthService(
		auth,
		NewMobileSessionIssuer(auth),
		NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute)),
		nil,
	)

	result, err := social.CompleteMobileLink(&model.User{
		USRSeq:    42,
		USRID:     "member",
		USRName:   "Member",
		USRStatus: "CCC",
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialAuthAuthenticated || result.Session == nil {
		t.Fatalf("result = %#v", result)
	}
	if result.Session.AccessToken == "" || result.Session.RefreshToken == "" {
		t.Fatal("completed mobile link must return both service tokens")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompletedPendingMobileSocialLinkReturnsPendingWithoutTokens(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	social := NewSocialAuthService(
		auth,
		NewMobileSessionIssuer(auth),
		NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute)),
		nil,
	)

	result, err := social.CompleteMobileLink(&model.User{
		USRSeq:    42,
		USRID:     "member",
		USRName:   "Member",
		USRStatus: "BBB",
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialAuthPending || result.Pending == nil || result.Session != nil {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshReadsLatestStatusAndRevokesForbiddenFamily(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	now := time.Now()
	token, _, _, err := auth.generateMobileRefreshToken(&model.AuthUser{
		USRSeq:    42,
		USRName:   "Member",
		USRStatus: "CCC",
	}, "family-1", now)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`FROM WEO_MEMBER`).
		WithArgs(42).
		WillReturnRows(memberRows("BBB"))
	mock.ExpectExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(42, "family-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err = NewMobileSessionIssuer(auth).Rotate(token)
	if !errors.Is(err, ErrLoginPending) {
		t.Fatalf("Rotate error = %v, want pending", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentRefreshAllowsOneRotationAndRevokesReplayFamily(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	auth.repo.DB.SetMaxOpenConns(2)
	mock.MatchExpectationsInOrder(false)
	now := time.Now()
	token, oldJTI, _, err := auth.generateMobileRefreshToken(&model.AuthUser{
		USRSeq:    42,
		USRName:   "Member",
		USRStatus: "CCC",
	}, "family-1", now)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		mock.ExpectQuery(`FROM WEO_MEMBER`).
			WithArgs(42).
			WillReturnRows(memberRows("CCC"))
		mock.ExpectBegin()
	}
	mock.ExpectQuery(`SELECT MRT_SID, EXPIRES_AT, CONSUMED_AT, REVOKED_AT`).
		WithArgs(oldJTI, 42).
		WillReturnRows(sqlmock.NewRows([]string{
			"MRT_SID", "EXPIRES_AT", "CONSUMED_AT", "REVOKED_AT",
		}).AddRow("family-1", now.Add(time.Hour), nil, nil))
	mock.ExpectExec(`SET CONSUMED_AT = NOW\(\), ROTATED_TO_JTI = \?`).
		WithArgs(sqlmock.AnyArg(), oldJTI, 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(sqlmock.AnyArg(), 42, "family-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	mock.ExpectQuery(`SELECT MRT_SID, EXPIRES_AT, CONSUMED_AT, REVOKED_AT`).
		WithArgs(oldJTI, 42).
		WillReturnRows(sqlmock.NewRows([]string{
			"MRT_SID", "EXPIRES_AT", "CONSUMED_AT", "REVOKED_AT",
		}).AddRow("family-1", now.Add(time.Hour), now, nil))
	mock.ExpectExec(`SET REVOKED_AT = COALESCE\(REVOKED_AT, NOW\(\)\)`).
		WithArgs(42, "family-1").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	issuer := NewMobileSessionIssuer(auth)
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			_, rotateErr := issuer.Rotate(token)
			results <- rotateErr
		}()
	}
	close(start)

	var succeeded int
	var replayed int
	for range 2 {
		rotateErr := <-results
		switch {
		case rotateErr == nil:
			succeeded++
		case errors.Is(rotateErr, repository.ErrRefreshTokenReplay):
			replayed++
		default:
			t.Fatalf("unexpected rotation error: %v", rotateErr)
		}
	}
	if succeeded != 1 || replayed != 1 {
		t.Fatalf("success=%d replay=%d", succeeded, replayed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSocialMemberIDKeepsKakaoCompatibilityAndBoundsAppleSubject(t *testing.T) {
	if got := socialMemberID(model.SocialProviderKakao, "12345"); got != "K12345" {
		t.Fatalf("Kakao member id = %q", got)
	}
	first := socialMemberID(model.SocialProviderApple, strings.Repeat("apple-subject", 50))
	second := socialMemberID(model.SocialProviderApple, strings.Repeat("apple-subject", 50))
	if first != second {
		t.Fatal("Apple member id must be deterministic")
	}
	if len(first) != 32 || !strings.HasPrefix(first, "SAP") {
		t.Fatalf("Apple member id = %q", first)
	}
}

func TestAppleNotificationsApplyAccountLifecycle(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		expect    func(sqlmock.Sqlmock)
	}{
		{
			name:      "consent revoked disconnects and ends sessions",
			eventType: "consent-revoked",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN`).
					WithArgs(42).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`DELETE FROM WEO_MEMBER_LOG`).
					WithArgs(42).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectBegin()
				mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
					WithArgs(42, "AP").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
					WithArgs(42, "AP").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:      "account deleted blocks login and removes link",
			eventType: "account-deleted",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE WEO_MEMBER SET USR_STATUS`).
					WithArgs("AAA", 42).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN`).
					WithArgs(42).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`DELETE FROM WEO_MEMBER_LOG`).
					WithArgs(42).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectBegin()
				mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
					WithArgs(42, "AP").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
					WithArgs(42, "AP").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:      "email disabled updates provider state",
			eventType: "email-disabled",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE WEO_MEMBER_SOCIAL`).
					WithArgs("ACTIVE", "N", 42, "AP").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:      "email enabled updates provider state",
			eventType: "email-enabled",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE WEO_MEMBER_SOCIAL`).
					WithArgs("ACTIVE", "Y", 42, "AP").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			auth, mock, cleanup := newAuthServiceForTest(t)
			defer cleanup()
			mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
				WithArgs("AP", "apple-subject").
				WillReturnRows(memberRows("CCC"))
			test.expect(mock)
			lifecycle := NewSocialAccountLifecycleService(auth, nil)

			err := lifecycle.ApplyAppleNotification(AppleServerNotification{
				Type:    test.eventType,
				Subject: "apple-subject",
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAccountDeletionRequiresImmediateDeactivation(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	mock.ExpectExec(`UPDATE WEO_MEMBER SET USR_STATUS`).
		WithArgs("AAA", 42).
		WillReturnError(errors.New("database unavailable"))
	lifecycle := NewSocialAccountLifecycleService(auth, nil)

	result, err := lifecycle.DeleteAccount(context.Background(), 42)

	if err == nil {
		t.Fatal("deactivation failure must reject account deletion")
	}
	if result.RevocationPending {
		t.Fatal("a rejected deletion must not be represented as accepted/pending")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountDeletionRecordsProviderRevocationAsPendingAfterDeactivation(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	mock.ExpectExec(`UPDATE WEO_MEMBER SET USR_STATUS`).
		WithArgs("AAA", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_LOG`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT NMS_GATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}).AddRow("AP"))
	mock.ExpectQuery(`SELECT ENCRYPTED_CREDENTIAL`).
		WithArgs(42, "AP").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(42, "AP", revocationActionDelete, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	lifecycle := NewSocialAccountLifecycleService(auth, nil)

	result, err := lifecycle.DeleteAccount(context.Background(), 42)

	if err != nil {
		t.Fatal(err)
	}
	if !result.RevocationPending {
		t.Fatal("provider revocation failure must be explicitly recoverable")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func newAuthServiceForTest(t *testing.T) (*AuthService, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
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
	auth := NewAuthService(
		authRepo,
		repository.NewSessionRepository(sqlxDB),
		cfg,
		cache.New(time.Minute, time.Minute),
		zerolog.Nop(),
	)
	return auth, mock, func() { _ = db.Close() }
}

func memberRows(status string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE",
		"USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
	}).AddRow(42, "member", "Member", status, nil, nil, nil, nil, nil)
}
