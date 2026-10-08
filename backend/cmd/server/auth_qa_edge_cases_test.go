// auth_qa_edge_cases_test.go — Exercises auth failure boundaries through real HTTP and MariaDB.
package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

func TestAuthQAAPIRecoveryBoundaries(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	t.Run("refresh_replay_revokes_only_its_family", func(t *testing.T) {
		_, first := s.login(t, "ios")
		_, other := s.login(t, "android")
		body := s.request(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refreshToken": first.RefreshToken}, "", "ios", "100", 200)
		rotated := decodeGolden[model.MobileSession](t, body)
		body = s.request(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refreshToken": first.RefreshToken}, "", "ios", "100", 401)
		assertGoldenError(t, body, "REFRESH_REPLAY_DETECTED")
		s.request(t, http.MethodGet, "/api/auth/me", nil, rotated.AccessToken, "ios", "100", 401)
		s.request(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refreshToken": rotated.RefreshToken}, "", "ios", "100", 401)
		s.request(t, http.MethodGet, "/api/auth/me", nil, other.AccessToken, "android", "100", 200)
		goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_SID=? AND REVOKED_AT IS NULL`, first.SID)
	})
	t.Run("current_logout_preserves_other_device_then_logout_all_revokes_it", func(t *testing.T) {
		_, first := s.login(t, "ios")
		_, other := s.login(t, "android")
		s.request(t, http.MethodPost, "/api/auth/logout", nil, first.AccessToken, "ios", "100", 204)
		s.request(t, http.MethodGet, "/api/auth/me", nil, first.AccessToken, "ios", "100", 401)
		s.request(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refreshToken": first.RefreshToken}, "", "ios", "100", 401)
		s.request(t, http.MethodGet, "/api/auth/me", nil, other.AccessToken, "android", "100", 200)
		s.request(t, http.MethodPost, "/api/auth/logout/all", nil, other.AccessToken, "android", "100", 204)
		s.request(t, http.MethodGet, "/api/auth/me", nil, other.AccessToken, "android", "100", 401)
		s.request(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refreshToken": other.RefreshToken}, "", "android", "100", 401)
		goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND REVOKED_AT IS NULL`, goldenMemberID)
	})
	t.Run("pending_login_and_refresh_keep_restricted_principal", func(t *testing.T) {
		goldenExec(t, s.db, `UPDATE WEO_MEMBER SET USR_STATUS='BBB' WHERE USR_SEQ=?`, goldenMemberID)
		goldenExec(t, s.db, `UPDATE ALUMNI_VERIFICATION SET STATUS='unsubmitted' WHERE USR_SEQ=?`, goldenMemberID)
		defer func() {
			goldenExec(t, s.db, `UPDATE WEO_MEMBER SET USR_STATUS='CCC' WHERE USR_SEQ=?`, goldenMemberID)
			goldenExec(t, s.db, `UPDATE ALUMNI_VERIFICATION SET STATUS='approved' WHERE USR_SEQ=?`, goldenMemberID)
		}()
		body := s.request(t, http.MethodPost, "/api/auth/mobile/login", map[string]string{"usrId": "golden_member", "password": goldenPassword}, "", "android", "100", 200)
		login := decodeGolden[model.SocialAuthResult](t, body)
		if login.Session == nil || login.Session.User.Verification.Status != model.VerificationUnsubmitted {
			t.Fatal("pending login must issue an unsubmitted principal")
		}
		s.request(t, http.MethodGet, "/api/auth/me", nil, login.Session.AccessToken, "android", "100", 200)
		body = s.request(t, http.MethodGet, "/api/alumni", nil, login.Session.AccessToken, "android", "100", 403)
		assertGoldenError(t, body, "ALUMNI_APPROVAL_REQUIRED")
		body = s.request(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refreshToken": login.Session.RefreshToken}, "", "android", "100", 200)
		rotated := decodeGolden[model.MobileSession](t, body)
		if rotated.User.Verification.Status != model.VerificationUnsubmitted {
			t.Fatal("refresh elevated the pending principal")
		}
		s.request(t, http.MethodGet, "/api/alumni", nil, rotated.AccessToken, "android", "100", 403)
	})
	t.Run("suspension_invalidates_existing_access_and_refresh", func(t *testing.T) {
		_, session := s.login(t, "ios")
		goldenExec(t, s.db, `UPDATE WEO_MEMBER SET USR_STATUS='DDD' WHERE USR_SEQ=?`, goldenMemberID)
		defer goldenExec(t, s.db, `UPDATE WEO_MEMBER SET USR_STATUS='CCC' WHERE USR_SEQ=?`, goldenMemberID)
		s.request(t, http.MethodGet, "/api/auth/me", nil, session.AccessToken, "ios", "100", 401)
		body := s.request(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refreshToken": session.RefreshToken}, "", "ios", "100", 403)
		assertGoldenError(t, body, "ACCOUNT_SUSPENDED")
		goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_SID=? AND REVOKED_AT IS NULL`, session.SID)
	})

	var verificationID, verificationToken string
	if !t.Run("password_signup_conflict_preserves_phone_grant_and_creates_no_session", func(t *testing.T) {
		body := s.request(t, http.MethodPost, "/api/auth/phone/verification/request", map[string]string{"phone": goldenPhone}, "", "ios", "100", 200)
		verificationID = decodeGolden[model.PhoneVerificationRequestResult](t, body).VerificationID
		body = s.request(t, http.MethodPost, "/api/auth/phone/verification/confirm", map[string]string{"verificationId": verificationID, "code": goldenCode}, "", "ios", "100", 200)
		verificationToken = decodeGolden[model.PhoneVerificationConfirmResult](t, body).VerificationToken
		body = s.request(t, http.MethodPost, "/api/auth/register", map[string]any{
			"usrId": "golden_member", "password": goldenPassword, "name": "Synthetic QA", "phone": goldenPhone,
			"email": "qa@example.test", "fn": "20", "fmDept": "영어", "phoneVerificationToken": verificationToken,
			"privacyConsent": map[string]any{"version": goldenConsentVersion, "accepted": true},
		}, "", "ios", "100", 409)
		assertGoldenError(t, body, "ID_TAKEN")
		goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_PHONE=?`, goldenPhone)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID=? AND CONSUMED_YN='N'`, verificationID)
	}) {
		return
	}
	linkStore := service.NewSocialLinkTokenStore(s.deps.cacheStore)
	const linkToken = "synthetic-auth-qa-social-continuation"
	const subject = "synthetic-auth-qa-kakao-subject"
	if _, err := linkStore.Put(linkToken, model.SocialLinkData{Provider: "KT", SocialID: subject, Email: "social@example.test"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	request := map[string]any{
		"token": linkToken, "mode": "new", "client": "mobile", "name": "Synthetic Social QA", "phone": goldenPhone,
		"email": "social@example.test", "fn": "20", "fmDept": "영어", "phoneVerificationToken": verificationToken,
		"privacyConsent": map[string]any{"version": goldenConsentVersion, "accepted": true},
	}
	t.Run("social_transaction_failure_preserves_grants_and_rolls_back_account", func(t *testing.T) {
		goldenExec(t, s.db, `CREATE TRIGGER auth_qa_fail_social_insert BEFORE INSERT ON WEO_MEMBER_SOCIAL FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='synthetic social write failure'`)
		defer goldenExec(t, s.db, `DROP TRIGGER auth_qa_fail_social_insert`)
		body := s.request(t, http.MethodPost, "/api/auth/social/link", request, "", "ios", "100", 500)
		assertGoldenError(t, body, "LINK_FAILED")
		goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_PHONE=?`, goldenPhone)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID=? AND CONSUMED_YN='N'`, verificationID)
		lease, err := linkStore.Begin(linkToken)
		if err != nil {
			t.Fatalf("rolled back signup must keep continuation retryable: %v", err)
		}
		if err := linkStore.Release(lease); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("social_session_issue_failure_leaves_no_service_session_and_recovers_by_login", func(t *testing.T) {
		goldenExec(t, s.db, `CREATE TRIGGER auth_qa_fail_session_insert BEFORE INSERT ON ALUMNI_MOBILE_REFRESH_TOKEN FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='synthetic session write failure'`)
		body := s.request(t, http.MethodPost, "/api/auth/social/link", request, "", "ios", "100", 500)
		goldenExec(t, s.db, `DROP TRIGGER auth_qa_fail_session_insert`)
		assertGoldenError(t, body, "LOGIN_FAILED")
		var seq int
		if err := s.db.Get(&seq, `SELECT USR_SEQ FROM WEO_MEMBER WHERE USR_PHONE=?`, goldenPhone); err != nil {
			t.Fatal(err)
		}
		goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=?`, seq)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID=? AND CONSUMED_YN='Y'`, verificationID)
		if _, err := linkStore.Begin(linkToken); !errors.Is(err, service.ErrSocialLinkTokenConsumed) {
			t.Fatalf("committed account must consume its continuation: %v", err)
		}
		body = s.request(t, http.MethodPost, "/api/auth/social/link", request, "", "ios", "100", 400)
		assertGoldenError(t, body, "PHONE_NOT_VERIFIED")
		// Provider verification is synthetic; all account lookup/session operations use the real DB.
		social := service.NewSocialAuthService(s.deps.authService, service.NewMobileSessionIssuer(s.deps.authService), linkStore, nil, authQASocialVerifier{subject: subject})
		recovered, err := social.Authenticate(context.Background(), model.KakaoAuthorization{AccessToken: "synthetic-provider-token"})
		if err != nil || recovered.Status != model.SocialAuthAuthenticated || recovered.Session == nil || recovered.Session.User.USRSeq != seq {
			t.Fatalf("fresh provider login must recover committed signup: err=%v status=%s", err, recovered.Status)
		}
		s.request(t, http.MethodGet, "/api/auth/me", nil, recovered.Session.AccessToken, "ios", "100", 200)
		s.request(t, http.MethodGet, "/api/alumni", nil, recovered.Session.AccessToken, "ios", "100", 403)
	})
	t.Run("deletion_cancel_requires_secret_and_keeps_old_sessions_revoked", func(t *testing.T) {
		_, first := s.login(t, "ios")
		_, other := s.login(t, "android")
		receipt, cancel := strings.Repeat("c7", 32), strings.Repeat("d7", 32)
		s.request(t, http.MethodPost, "/api/auth/account/deletion-requests", map[string]string{"receiptToken": receipt, "cancelToken": cancel}, first.AccessToken, "ios", "100", 202)
		s.request(t, http.MethodGet, "/api/auth/me", nil, first.AccessToken, "ios", "100", 401)
		s.request(t, http.MethodGet, "/api/auth/me", nil, other.AccessToken, "android", "100", 401)
		s.request(t, http.MethodPost, "/api/auth/mobile/login", map[string]string{"usrId": "golden_member", "password": goldenPassword}, "", "ios", "100", 403)
		s.request(t, http.MethodPost, "/api/account-deletion/cancel", map[string]string{"receiptToken": receipt, "cancelToken": strings.Repeat("e7", 32)}, "", "ios", "100", 404)
		body := s.request(t, http.MethodPost, "/api/account-deletion/cancel", map[string]string{"receiptToken": receipt, "cancelToken": cancel}, "", "ios", "100", 200)
		if decodeGolden[model.AccountDeletionReceipt](t, body).Status != "cancelled" {
			t.Fatal("correct cancellation credentials must restore the pending account")
		}
		s.request(t, http.MethodGet, "/api/auth/me", nil, first.AccessToken, "ios", "100", 401)
		s.request(t, http.MethodGet, "/api/auth/me", nil, other.AccessToken, "android", "100", 401)
		s.login(t, "ios")
	})
}

type authQASocialVerifier struct{ subject string }

func (authQASocialVerifier) Provider() model.SocialProvider { return model.SocialProviderKakao }

func (v authQASocialVerifier) Verify(context.Context, model.SocialAuthorization) (service.VerifiedSocialAccount, error) {
	return service.VerifiedSocialAccount{Identity: model.VerifiedSocialIdentity{Provider: model.SocialProviderKakao, Subject: v.subject}}, nil
}
