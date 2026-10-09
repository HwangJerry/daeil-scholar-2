package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

func TestSignupEvidenceFailureRollsBackAndCanRetry(t *testing.T) {
	for _, kind := range []string{"password", "social"} {
		for _, fault := range []string{"consent", "phone_grant"} {
			t.Run(kind+"/"+fault, func(t *testing.T) {
				s := newGoldenServer(t)
				b := s.request(t, http.MethodPost, "/api/auth/phone/verification/request", map[string]string{"phone": goldenPhone}, "", "android", "100", 200)
				id := decodeGolden[model.PhoneVerificationRequestResult](t, b).VerificationID
				b = s.request(t, http.MethodPost, "/api/auth/phone/verification/confirm", map[string]string{"verificationId": id, "code": goldenCode}, "", "android", "100", 200)
				grant := decodeGolden[model.PhoneVerificationConfirmResult](t, b).VerificationToken
				body := map[string]any{"usrId": "atomicqa", "password": goldenPassword, "name": "Synthetic Atomic QA", "phone": goldenPhone, "email": "atomic@example.test", "fn": "20", "fmDept": "영어", "phoneVerificationToken": grant, "privacyConsent": map[string]any{"version": goldenConsentVersion, "accepted": true}}
				path, success := "/api/auth/register", http.StatusCreated
				if kind == "social" {
					store := service.NewSocialLinkTokenStore(s.deps.cacheStore)
					_, err := store.Put("atomic-qa-link", model.SocialLinkData{Provider: "KT", SocialID: "atomic-subject", Email: "atomic@example.test"}, time.Minute)
					if err != nil {
						t.Fatal(err)
					}
					body["token"], body["mode"], body["client"] = "atomic-qa-link", "new", "mobile"
					path, success = "/api/auth/social/link", http.StatusOK
				}
				trigger := "qa_atomic_fail_" + fault
				if fault == "consent" {
					goldenExec(t, s.db, `CREATE TRIGGER `+trigger+` BEFORE INSERT ON AUTH_CONSENT FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='synthetic consent failure'`)
				} else {
					goldenExec(t, s.db, `CREATE TRIGGER `+trigger+` BEFORE UPDATE ON ALUMNI_PHONE_VERIFICATION FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='synthetic phone grant failure'`)
				}
				s.request(t, http.MethodPost, path, body, "", "android", "100", 500)
				goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_PHONE=?`, goldenPhone)
				goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM AUTH_PHONE_CLAIM WHERE CANONICAL_PHONE=?`, goldenPhone)
				goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM AUTH_CONSENT`)
				goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM AUTH_IDENTITY`)
				goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN`)
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID=? AND CONSUMED_YN='N' AND CONSUMED_USR_SEQ IS NULL`, id)
				goldenExec(t, s.db, `DROP TRIGGER `+trigger)
				s.request(t, http.MethodPost, path, body, "", "android", "100", success)
				var user int
				if err := s.db.Get(&user, `SELECT USR_SEQ FROM WEO_MEMBER WHERE USR_PHONE=?`, goldenPhone); err != nil {
					t.Fatal(err)
				}
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_CONSENT WHERE ACCOUNT_ID=? AND IS_ACCEPTED=1`, user)
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID=? AND CONSUMED_YN='Y' AND CONSUMED_USR_SEQ=? AND CONSUMED_AT IS NOT NULL`, id, user)
			})
		}
	}
}

func TestMobileLoginErrorNamesSelectedIdentifier(t *testing.T) {
	s := newGoldenServer(t)
	for _, c := range []struct {
		name    string
		body    map[string]string
		status  int
		message string
	}{
		{"unknown_id", map[string]string{"usrId": "unknown_atomic_qa", "password": "wrong"}, 401, "아이디 또는 비밀번호가 올바르지 않습니다"},
		{"missing_id", map[string]string{"usrId": "", "password": "wrong"}, 400, "아이디와 비밀번호를 입력해주세요"},
		{"missing_password_for_email", map[string]string{"email": "synthetic@example.test"}, 400, "이메일과 비밀번호를 입력해주세요"},
		{"unknown_email", map[string]string{"email": "synthetic@example.test", "password": "wrong"}, 401, "이메일 또는 비밀번호가 올바르지 않습니다"},
	} {
		t.Run(c.name, func(t *testing.T) {
			b := s.request(t, http.MethodPost, "/api/auth/mobile/login", c.body, "", "android", "100", c.status)
			got := decodeGolden[struct {
				Message string `json:"message"`
			}](t, b)
			if got.Message != c.message {
				t.Fatalf("message=%q, want %q", got.Message, c.message)
			}
		})
	}
}

func TestSignupRechecksPhoneGrantInsideMemberTransaction(t *testing.T) {
	for _, social := range []bool{false, true} {
		t.Run(map[bool]string{false: "password", true: "social"}[social], func(t *testing.T) {
			s := newGoldenServer(t)
			b := s.request(t, http.MethodPost, "/api/auth/phone/verification/request", map[string]string{"phone": goldenPhone}, "", "android", "100", 200)
			id := decodeGolden[model.PhoneVerificationRequestResult](t, b).VerificationID
			b = s.request(t, http.MethodPost, "/api/auth/phone/verification/confirm", map[string]string{"verificationId": id, "code": goldenCode}, "", "android", "100", 200)
			grant := decodeGolden[model.PhoneVerificationConfirmResult](t, b).VerificationToken
			body := map[string]any{"usrId": "expiringqa", "password": goldenPassword, "name": "Synthetic Expiry QA", "phone": goldenPhone, "email": "expiry@example.test", "fn": "20", "fmDept": "영어", "phoneVerificationToken": grant, "privacyConsent": map[string]any{"version": goldenConsentVersion, "accepted": true}}
			path, success := "/api/auth/register", http.StatusCreated
			if social {
				store := service.NewSocialLinkTokenStore(s.deps.cacheStore)
				_, err := store.Put("expiry-qa-link", model.SocialLinkData{Provider: "KT", SocialID: "expiry-subject", Email: "expiry@example.test"}, time.Minute)
				if err != nil {
					t.Fatal(err)
				}
				body["token"], body["client"] = "expiry-qa-link", "mobile"
				path, success = "/api/auth/social/link", http.StatusOK
			}
			// The initial service assertion succeeds; expiry changes only during the
			// member insert. The transaction must check usability again before commit.
			goldenExec(t, s.db, `CREATE TRIGGER qa_expire_grant_at_member_insert BEFORE INSERT ON WEO_MEMBER FOR EACH ROW UPDATE ALUMNI_PHONE_VERIFICATION SET GRANT_EXPIRES_AT=DATE_SUB(NOW(),INTERVAL 1 SECOND)`)
			b = s.request(t, http.MethodPost, path, body, "", "android", "100", 400)
			assertGoldenError(t, b, "PHONE_NOT_VERIFIED")
			goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_PHONE=?`, goldenPhone)
			goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM AUTH_CONSENT`)
			goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID=? AND CONSUMED_YN='N' AND GRANT_EXPIRES_AT>NOW()`, id)
			goldenExec(t, s.db, `DROP TRIGGER qa_expire_grant_at_member_insert`)
			s.request(t, http.MethodPost, path, body, "", "android", "100", success)
		})
	}
}
