package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
)

type socialOwnershipProviderTransport func(*http.Request) (*http.Response, error)

func (f socialOwnershipProviderTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestSocialSignupOwnershipConflictAfterContinuationRollsBackHTTP(t *testing.T) {
	for _, identityStatus := range []string{"ACTIVE", "REVOKED"} {
		t.Run(identityStatus, func(t *testing.T) {
			s := newGoldenServer(t)
			seedGoldenMember(t, s.db)
			const subject = "synthetic-late-ownership"
			store := service.NewSocialLinkTokenStore(s.deps.cacheStore)
			social := service.NewSocialAuthService(s.deps.authService, nil, store, nil, authQASocialVerifier{subject: subject})
			// Provider proof is fresh and unowned when the continuation is issued.
			result, err := social.Authenticate(context.Background(), model.KakaoAuthorization{AccessToken: "synthetic-provider-proof"})
			if err != nil || result.Status != model.SocialAuthLinkRequired || result.LinkRequired == nil {
				t.Fatalf("unowned provider did not issue continuation: %v", err)
			}
			token := result.LinkRequired.LinkToken
			// A different request establishes canonical ownership before submit.
			if err := s.deps.authRepo.LinkSocialIdentity(repository.SocialAccountFields{USRSeq: goldenMemberID, Provider: "KT", SocialID: subject}); err != nil {
				t.Fatal(err)
			}
			if identityStatus == "REVOKED" {
				if err := s.deps.authRepo.DeleteSocialConnection(goldenMemberID, "KT"); err != nil {
					t.Fatal(err)
				}
			}
			b := s.request(t, http.MethodPost, "/api/auth/phone/verification/request", map[string]string{"phone": goldenPhone}, "", "android", "100", http.StatusOK)
			id := decodeGolden[model.PhoneVerificationRequestResult](t, b).VerificationID
			b = s.request(t, http.MethodPost, "/api/auth/phone/verification/confirm", map[string]string{"verificationId": id, "code": goldenCode}, "", "android", "100", http.StatusOK)
			grant := decodeGolden[model.PhoneVerificationConfirmResult](t, b).VerificationToken
			request := map[string]any{"token": token, "mode": "new", "client": "mobile", "name": "Synthetic final ownership", "email": "synthetic@example.test", "phone": goldenPhone, "fn": "20", "fmDept": "영어", "phoneVerificationToken": grant, "privacyConsent": map[string]any{"version": goldenConsentVersion, "accepted": true}}
			for range 2 {
				b = s.request(t, http.MethodPost, "/api/auth/social/link", request, "", "android", "100", http.StatusConflict)
				assertGoldenError(t, b, "SOCIAL_ACCOUNT_ALREADY_LINKED")
				rejection := decodeGolden[model.SocialAuthRejection](t, b)
				if rejection.Message != "이미 연결된 소셜 계정입니다. 기존 로그인 방법으로 로그인한 뒤 계정 설정에서 소셜 계정 연결을 확인해주세요." {
					t.Fatal("final ownership conflict did not provide accurate recovery")
				}
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_MEMBER`)
				goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_PHONE=?`, goldenPhone)
				goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM AUTH_PHONE_CLAIM WHERE CANONICAL_PHONE=?`, goldenPhone)
				goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM AUTH_CONSENT`)
				goldenCount(t, s.db, 2, `SELECT COUNT(*) FROM AUTH_IDENTITY`)
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_ACCOUNT_STATE`)
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_VERIFICATION`)
				goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN`)
				goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_SOCIAL_CREDENTIAL`)
				legacyConnections := 0
				if identityStatus == "ACTIVE" {
					legacyConnections = 1
				}
				goldenCount(t, s.db, legacyConnections, `SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL`)
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID=? AND CONSUMED_YN='N' AND CONSUMED_USR_SEQ IS NULL`, id)
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_IDENTITY WHERE ACCOUNT_ID=? AND PROVIDER='KAKAO' AND SUBJECT_KEY=? AND STATUS=?`, goldenMemberID, subject, identityStatus)
				if _, err := store.Snapshot(token); err != nil {
					t.Fatalf("failed transaction consumed continuation: %v", err)
				}
			}
			s.request(t, http.MethodPost, "/api/auth/social/link/cancel", map[string]string{"token": token}, "", "android", "100", http.StatusNoContent)
			if _, err := store.Begin(token); err != service.ErrSocialLinkTokenCancelled {
				t.Fatalf("conflicting continuation could not be cancelled: %v", err)
			}
		})
	}
}

// Only provider proof is synthetic. The HTTP handler, canonical ownership,
// explicit relinking and session issuance use the disposable production schema.
func TestSocialAuthReservedOwnershipRejectsBeforeSignupHTTP(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	const subject = "880021"
	fields := repository.SocialAccountFields{USRSeq: goldenMemberID, Provider: "KT", SocialID: subject}
	if err := s.deps.authRepo.LinkSocialIdentity(fields); err != nil {
		t.Fatal(err)
	}
	if err := s.deps.authRepo.DeleteSocialConnection(goldenMemberID, "KT"); err != nil {
		t.Fatal(err)
	}
	transport := http.DefaultTransport
	providerCalls := 0
	http.DefaultTransport = socialOwnershipProviderTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || r.URL.Host != "kapi.kakao.com" || r.URL.Path != "/v2/user/me" {
			return nil, fmt.Errorf("unexpected provider request in isolated test")
		}
		providerCalls++
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":880021,"kakao_account":{"email":"synthetic@example.test"}}`)), Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = transport })
	continuationStore := service.NewSocialLinkTokenStore(s.deps.cacheStore)
	body := s.request(t, http.MethodPost, "/api/auth/kakao/mobile", map[string]string{"grantType": "access_token", "accessToken": "synthetic-provider-proof"}, "", "android", "100", http.StatusForbidden)
	assertGoldenError(t, body, "SOCIAL_ACCOUNT_RELINK_REQUIRED")
	var rejection model.SocialAuthRejection
	if err := json.Unmarshal(body, &rejection); err != nil {
		t.Fatal(err)
	}
	if rejection.Message != "기존 로그인 방법으로 로그인한 뒤 계정 설정에서 같은 소셜 계정을 다시 연결해주세요." {
		t.Fatal("HTTP recovery message must support accounts without ID/password login")
	}
	for _, forbidden := range []string{subject, "synthetic-provider-proof", "synthetic@example.test", "accessToken", "refreshToken", "linkToken", "usrId", "usrSeq"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("rejection exposed identity or continuation: %s", forbidden)
		}
	}
	if providerCalls != 1 {
		t.Fatal("ownership decision was made without fresh provider verification")
	}
	for key := range s.deps.cacheStore.Items() {
		if _, err := continuationStore.Snapshot(key); err == nil {
			t.Fatal("rejection created a continuation")
		}
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_MEMBER`)
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN`)
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_SOCIAL_CREDENTIAL`)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_IDENTITY WHERE PROVIDER='KAKAO' AND SUBJECT_KEY=? AND ACCOUNT_ID=? AND STATUS='REVOKED'`, subject, goldenMemberID)

	// The rejection must preserve the supported recovery path for the same owner.
	s.login(t, "android")
	social := service.NewSocialAuthService(s.deps.authService, service.NewMobileSessionIssuer(s.deps.authService), service.NewSocialLinkTokenStore(s.deps.cacheStore), nil, authQASocialVerifier{subject: subject})
	if _, err := social.LinkIdentity(context.Background(), goldenMemberID, "kakao", model.KakaoAuthorization{AccessToken: "synthetic-provider-proof"}); err != nil {
		t.Fatal(err)
	}
	result, err := social.Authenticate(context.Background(), model.KakaoAuthorization{AccessToken: "synthetic-provider-proof"})
	if err != nil || result.Status != model.SocialAuthAuthenticated || result.Session == nil || result.Session.User.USRSeq != goldenMemberID {
		t.Fatalf("same-owner relink did not restore login: %v", err)
	}
	s.request(t, http.MethodGet, "/api/auth/me", nil, result.Session.AccessToken, "android", "100", http.StatusOK)

	// After actual removal of the canonical ownership, this provider is new again.
	if err := s.deps.authRepo.DeleteSocialConnection(goldenMemberID, "KT"); err != nil {
		t.Fatal(err)
	}
	goldenExec(t, s.db, `DELETE FROM AUTH_IDENTITY WHERE ACCOUNT_ID=? AND PROVIDER='KAKAO'`, goldenMemberID)
	result, err = social.Authenticate(context.Background(), model.KakaoAuthorization{AccessToken: "synthetic-provider-proof"})
	if err != nil || result.Status != model.SocialAuthLinkRequired || result.LinkRequired == nil {
		t.Fatalf("removed ownership did not permit fresh signup: %v", err)
	}
}
