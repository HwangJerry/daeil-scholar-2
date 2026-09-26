package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/golden"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
)

func TestGoldenPublicSettings(t *testing.T) {
	s := newGoldenServer(t)
	for _, platform := range []string{"ios", "android"} {
		seedGoldenPolicy(t, s, platform)
	}
	goldenExec(t, s.db, `INSERT INTO app_settings (AS_KEY, AS_VALUE, AS_PUBLIC, UPDATED_AT)
		VALUES ('synthetic_private_setting', 'not-public', 'N', NOW())`)
	body := s.request(t, http.MethodGet, "/api/settings/public", nil, "", "ios", "100", http.StatusOK)
	settings := decodeGolden[map[string]string](t, body)
	if len(settings) != 2 {
		t.Fatalf("public settings must expose only the two policies: %s", body)
	}
	for _, platform := range []string{"ios", "android"} {
		policy := decodeGolden[model.AppUpdatePolicy](t, []byte(settings["app_update_policy_"+platform]))
		if !policy.ForceEnabled || policy.MinBuild != 100 {
			t.Fatalf("missing %s force policy", platform)
		}
	}
	golden.Assert(t, "settings_public", body) // No volatile fields; policies remain JSON strings.
}

func TestGoldenSignup(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	t.Run("check_id_available", func(t *testing.T) {
		body := s.request(t, http.MethodGet, "/api/auth/check-id?usrId=golden_signup", nil, "", "ios", "100", http.StatusOK)
		if !decodeGolden[map[string]bool](t, body)["available"] {
			t.Fatal("synthetic signup ID should be available")
		}
		golden.Assert(t, "check_id_available", body)
	})
	t.Run("check_phone_taken", func(t *testing.T) {
		body := s.request(t, http.MethodGet, "/api/auth/check-phone?phone=01000000002", nil, "", "ios", "100", http.StatusOK)
		if decodeGolden[map[string]bool](t, body)["available"] {
			t.Fatal("seeded member's phone should be taken")
		}
		golden.Assert(t, "check_phone_taken", body)
	})
	var verificationID, verificationToken string
	if !t.Run("phone_verification_request", func(t *testing.T) {
		body := s.request(t, http.MethodPost, "/api/auth/phone/verification/request", map[string]string{"phone": goldenPhone}, "", "ios", "100", http.StatusOK)
		result := decodeGolden[model.PhoneVerificationRequestResult](t, body)
		if result.VerificationID == "" || result.ExpiresInSec != 300 {
			t.Fatalf("invalid verification request result: %s", body)
		}
		verificationID = result.VerificationID
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION
			WHERE APV_ID = ? AND PHONE = ? AND VERIFIED_YN = 'N' AND CONSUMED_YN = 'N'`, verificationID, goldenPhone)
		golden.Assert(t, "phone_verification_request", body, "verificationId")
	}) {
		return
	}
	if !t.Run("phone_verification_confirm", func(t *testing.T) {
		body := s.request(t, http.MethodPost, "/api/auth/phone/verification/confirm", map[string]string{"verificationId": verificationID, "code": goldenCode}, "", "ios", "100", http.StatusOK)
		result := decodeGolden[model.PhoneVerificationConfirmResult](t, body)
		if result.VerificationToken == "" || result.ExpiresInSec != 1800 {
			t.Fatalf("invalid verification grant: %s", body)
		}
		verificationToken = result.VerificationToken
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION
			WHERE APV_ID = ? AND VERIFIED_YN = 'Y' AND CONSUMED_YN = 'N' AND GRANT_TOKEN_HASH = ?`, verificationID, goldenHash(verificationToken))
		golden.Assert(t, "phone_verification_confirm", body) // verificationToken is normalized by the token rule.
	}) {
		return
	}
	request := map[string]any{
		"usrId": "golden_member", "password": goldenPassword, "name": "합성 가입자", "phone": goldenPhone,
		"email": "signup@example.test", "fn": "20", "fmDept": "영어", "phoneVerificationToken": verificationToken,
		"privacyConsent": map[string]any{"version": goldenConsentVersion, "accepted": true},
	}
	t.Run("register_id_taken", func(t *testing.T) {
		body := s.request(t, http.MethodPost, "/api/auth/register", request, "", "ios", "100", http.StatusConflict)
		assertGoldenError(t, body, "ID_TAKEN")
		goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_PHONE = ?`, goldenPhone)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID = ? AND CONSUMED_YN = 'N'`, verificationID)
		golden.Assert(t, "register_id_taken", body)
	})
	t.Run("register_success", func(t *testing.T) {
		request["usrId"] = "golden_signup"
		body := s.request(t, http.MethodPost, "/api/auth/register", request, "", "ios", "100", http.StatusCreated)
		user := decodeGolden[model.AuthUser](t, body)
		if user.USRSeq <= 0 || user.USRID != "golden_signup" || user.USRStatus != "BBB" {
			t.Fatalf("signup must create a pending member: %s", body)
		}
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ = ? AND USR_STATUS = 'BBB' AND USR_PHONE = ?`, user.USRSeq, goldenPhone)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_PHONE_CLAIM WHERE ACCOUNT_ID = ? AND CANONICAL_PHONE = ?`, user.USRSeq, goldenPhone)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_ACCOUNT_STATE WHERE ACCOUNT_ID = ? AND STATUS = 'ACTIVE'`, user.USRSeq)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_VERIFICATION WHERE USR_SEQ = ? AND STATUS = 'unsubmitted'`, user.USRSeq)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_IDENTITY i JOIN AUTH_PASSWORD_CREDENTIAL c ON c.IDENTITY_ID = i.IDENTITY_ID
			WHERE i.ACCOUNT_ID = ? AND i.PROVIDER = 'LOCAL_USERNAME' AND c.ALGORITHM = 'ARGON2ID' AND c.STATUS = 'ACTIVE'`, user.USRSeq)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_CONSENT WHERE ACCOUNT_ID = ? AND CONSENT_TYPE = 'PRIVACY'
			AND CONSENT_VERSION = ? AND IS_ACCEPTED = 1 AND IS_REQUIRED = 1`, user.USRSeq, goldenConsentVersion)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID = ? AND CONSUMED_YN = 'Y'`, verificationID)
		golden.Assert(t, "register_success", body) // Sequence is deterministic in this fresh, seeded database.
	})
}

func TestGoldenMobileSession(t *testing.T) {
	mariadb.Start(t)
	// Both apps use identical login/refresh JSON; exercise each release header shape
	// against its own router/cache/database and compare with the same fixtures.
	for _, platform := range []string{"ios", "android"} {
		t.Run(platform, func(t *testing.T) {
			s := newGoldenServer(t)
			seedGoldenMember(t, s.db)
			var session model.MobileSession
			if !t.Run("mobile_login_success", func(t *testing.T) {
				body, issued := s.login(t, platform)
				session = issued
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ = ? AND MRT_JTI = ? AND MRT_SID = ? AND REVOKED_AT IS NULL`, goldenMemberID, session.JTI, session.SID)
				golden.Assert(t, "mobile_login_success", body, "sid", "jti", "accessIssuedAt", "accessExpiresAt", "refreshExpiresAt")
			}) {
				return
			}
			t.Run("mobile_login_invalid_password", func(t *testing.T) {
				body := s.request(t, http.MethodPost, "/api/auth/mobile/login", map[string]string{"usrId": "golden_member", "password": "wrong-synthetic-password"}, "", platform, "100", http.StatusUnauthorized)
				assertGoldenError(t, body, "INVALID_CREDENTIALS")
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ = ?`, goldenMemberID)
				golden.Assert(t, "mobile_login_invalid_password", body)
			})
			if !t.Run("refresh_success", func(t *testing.T) {
				body := s.request(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refreshToken": session.RefreshToken}, "", platform, "100", http.StatusOK)
				rotated := decodeGolden[model.MobileSession](t, body)
				assertGoldenSession(t, rotated)
				if rotated.RefreshToken == session.RefreshToken || rotated.JTI == session.JTI || rotated.SID != session.SID {
					t.Fatal("refresh must rotate the token/JTI within the same session")
				}
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN
					WHERE MRT_JTI = ? AND CONSUMED_AT IS NOT NULL AND ROTATED_TO_JTI = ?`, session.JTI, rotated.JTI)
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN
					WHERE MRT_JTI = ? AND MRT_SID = ? AND CONSUMED_AT IS NULL AND REVOKED_AT IS NULL`, rotated.JTI, session.SID)
				session = rotated
				golden.Assert(t, "refresh_success", body, "sid", "jti", "accessIssuedAt", "accessExpiresAt", "refreshExpiresAt")
			}) {
				return
			}
			t.Run("auth_me", func(t *testing.T) {
				body := s.request(t, http.MethodGet, "/api/auth/me", nil, session.AccessToken, platform, "100", http.StatusOK)
				user := decodeGolden[model.AuthUser](t, body)
				if user.USRSeq != goldenMemberID || user.Email != "member@example.test" || user.Verification.Status != model.VerificationApproved || user.AdminRole != nil {
					t.Fatalf("wrong authenticated principal: %s", body)
				}
				golden.Assert(t, "auth_me", body) // submittedAt/reviewedAt use automatic timestamp normalization.
			})
			t.Run("app_update_required_426", func(t *testing.T) {
				seedGoldenPolicy(t, s, platform)
				body := s.request(t, http.MethodGet, "/api/auth/me", nil, session.AccessToken, platform, "99", http.StatusUpgradeRequired)
				assertGoldenError(t, body, "APP_UPDATE_REQUIRED")
				golden.Assert(t, "app_update_required_426", body)
				// The same valid session is admitted at the minimum build.
				s.request(t, http.MethodGet, "/api/auth/me", nil, session.AccessToken, platform, "100", http.StatusOK)
			})
		})
	}
}

func TestGoldenAccountDeletion(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db) // Disposable member in this test's database only.
	_, session := s.login(t, "android")
	receiptToken, cancelToken := strings.Repeat("a7", 32), strings.Repeat("b7", 32)
	var receipt model.AccountDeletionReceipt
	if !t.Run("account_deletion_request", func(t *testing.T) {
		body := s.request(t, http.MethodPost, "/api/auth/account/deletion-requests", map[string]string{
			"receiptToken": receiptToken, "cancelToken": cancelToken,
		}, session.AccessToken, "android", "100", http.StatusAccepted)
		result := decodeGolden[struct {
			Receipt               model.AccountDeletionReceipt `json:"receipt"`
			ReceiptToken          string                       `json:"receiptToken"`
			SessionCleanupPending bool                         `json:"sessionCleanupPending"`
		}](t, body)
		receipt = result.Receipt
		if receipt.ID <= 0 || receipt.Status != "pending" || !receipt.CanCancel || result.ReceiptToken != receiptToken || result.SessionCleanupPending {
			t.Fatalf("deletion must accept a cancellable receipt: %s", body)
		}
		if receipt.TargetAt.Sub(receipt.RequestedAt) != 24*time.Hour || receipt.ScheduledAt == nil || !receipt.ScheduledAt.Equal(receipt.TargetAt) {
			t.Fatal("deletion must honor the configured wait period")
		}
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ = ? AND USR_STATUS = 'AAA'`, goldenMemberID)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID = ? AND USR_SEQ = ? AND RECEIPT_HASH = ?`, receipt.ID, goldenMemberID, goldenDeletionHash(t, receiptToken))
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ERASURE_CANCELLATION WHERE REQUEST_ID = ? AND CANCEL_HASH = ?`, receipt.ID, goldenDeletionHash(t, cancelToken))
		goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ = ? AND REVOKED_AT IS NULL`, goldenMemberID)
		golden.Assert(t, "account_deletion_request", body) // Fixed ID; receiptToken/timestamps normalize automatically.
	}) {
		return
	}
	t.Run("account_deletion_receipt", func(t *testing.T) {
		// Both native apps POST the receipt token, with no Authorization header.
		body := s.request(t, http.MethodPost, "/api/account-deletion/receipt", map[string]string{"receiptToken": receiptToken}, "", "android", "100", http.StatusOK)
		found := decodeGolden[model.AccountDeletionReceipt](t, body)
		if found.ID != receipt.ID || found.Status != "pending" || !found.CanCancel || !found.RequestedAt.Equal(receipt.RequestedAt) {
			t.Fatalf("receipt lookup must recover the accepted request: %s", body)
		}
		golden.Assert(t, "account_deletion_receipt", body)
	})
}

func seedGoldenPolicy(t *testing.T, s *goldenServer, platform string) {
	t.Helper()
	policy := model.AppUpdatePolicy{ForceEnabled: true, MinBuild: 100, RecommendEnabled: true, RecommendedBuild: 110, MinOSVersion: "1.0", StoreURL: "https://store.example.test/" + platform}
	value, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	goldenExec(t, s.db, `INSERT INTO app_settings (AS_KEY, AS_VALUE, AS_PUBLIC, UPDATED_AT) VALUES (?, ?, 'Y', NOW())`, "app_update_policy_"+platform, string(value))
	s.deps.cacheStore.Flush() // Direct fixture SQL must invalidate the real public-settings cache.
}

func assertGoldenError(t *testing.T, body []byte, code string) {
	t.Helper()
	err := decodeGolden[model.APIError](t, body)
	if err.Code != code || err.Message == "" {
		t.Fatalf("expected %s error envelope, got %s", code, body)
	}
}

func goldenHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func goldenDeletionHash(t *testing.T, token string) string {
	t.Helper()
	decoded, err := hex.DecodeString(token)
	if err != nil {
		t.Fatal(err)
	}
	return goldenHash(string(decoded))
}
