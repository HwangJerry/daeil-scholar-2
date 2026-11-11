package main

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"net/http"
	"testing"
	"time"
)

func TestSignupPendingAvailabilityUsesCanonicalStatusWithoutCreatingAccount(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	cases := []struct {
		status, lifecycle, memberStatus string
		pending                         bool
	}{
		{"pending", "ACTIVE", "CCC", true},
		{"reapproval_pending", "ACTIVE", "CCC", true},
		{"approved", "ACTIVE", "CCC", false},
		{"unsubmitted", "ACTIVE", "CCC", false},
		{"rejected", "ACTIVE", "CCC", false},
		{"pending", "WITHDRAWN", "AAA", false},
	}
	for _, c := range cases {
		t.Run(c.status+"-"+c.lifecycle, func(t *testing.T) {
			goldenExec(t, s.db, "UPDATE ALUMNI_VERIFICATION SET STATUS=? WHERE USR_SEQ=?", c.status, goldenMemberID)
			goldenExec(t, s.db, "UPDATE AUTH_ACCOUNT_STATE SET STATUS=? WHERE ACCOUNT_ID=?", c.lifecycle, goldenMemberID)
			goldenExec(t, s.db, "UPDATE WEO_MEMBER SET USR_STATUS=? WHERE USR_SEQ=?", c.memberStatus, goldenMemberID)
			for _, path := range []string{"/api/auth/check-id?usrId=golden_member", "/api/auth/check-phone?phone=01000000002", "/api/auth/check-email?email=member%40example.test"} {
				result := decodeGolden[map[string]bool](t, s.request(t, http.MethodGet, path, nil, "", "ios", "100", 200))
				if result["available"] || result["signupPending"] != c.pending {
					t.Fatalf("%s: %v", path, result)
				}
			}
		})
	}
	goldenCount(t, s.db, 1, "SELECT COUNT(*) FROM WEO_MEMBER")
	goldenCount(t, s.db, 0, "SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN")
}

func TestRepeatedSocialSignupReturnsPendingNoticeWithoutSessionOrAccount(t *testing.T) {
	s := newGoldenServer(t)
	member := defaultGoldenMember
	member.phone = goldenPhone
	seedGoldenMemberAs(t, s.db, member)
	goldenExec(t, s.db, "UPDATE ALUMNI_VERIFICATION SET STATUS='pending' WHERE USR_SEQ=?", goldenMemberID)
	store := service.NewSocialLinkTokenStore(s.deps.cacheStore)
	_, err := store.Put("pending-retry", model.SocialLinkData{Provider: "KT", SocialID: "synthetic-new-provider"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	sent := s.request(t, http.MethodPost, "/api/auth/phone/verification/request", map[string]string{"phone": goldenPhone}, "", "ios", "100", 200)
	id := decodeGolden[model.PhoneVerificationRequestResult](t, sent).VerificationID
	confirmed := s.request(t, http.MethodPost, "/api/auth/phone/verification/confirm", map[string]string{"verificationId": id, "code": goldenCode}, "", "ios", "100", 200)
	grant := decodeGolden[model.PhoneVerificationConfirmResult](t, confirmed).VerificationToken
	body := map[string]any{"token": "pending-retry", "mode": "new", "client": "mobile", "name": "Synthetic", "email": "new@example.test", "phone": goldenPhone, "fn": "20", "fmDept": "영어", "phoneVerificationToken": grant, "privacyConsent": map[string]any{"version": goldenConsentVersion, "accepted": true}}
	for range 2 {
		raw := s.request(t, http.MethodPost, "/api/auth/social/link", body, "", "ios", "100", 409)
		assertGoldenError(t, raw, "SIGNUP_ALREADY_PENDING")
		notice := decodeGolden[model.SocialAuthRejection](t, raw)
		if notice.Message != "가입 요청이 정상적으로 처리되었습니다. 검토 후 승인됩니다." {
			t.Fatal(notice.Message)
		}
	}
	goldenCount(t, s.db, 1, "SELECT COUNT(*) FROM WEO_MEMBER")
	goldenCount(t, s.db, 0, "SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN")
}
