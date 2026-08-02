package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestKakaoResponseUsesCanonicalSessionEnvelopeOnly(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeMobileAuthResult(recorder, model.SocialAuthResult{
		Status: model.SocialAuthAuthenticated,
		Session: &model.MobileSession{
			User: model.AuthUser{
				USRSeq:    42,
				USRID:     "member",
				USRName:   "Member",
				USRStatus: "CCC",
			},
			AccessToken:      "access",
			RefreshToken:     "refresh",
			AccessIssuedAt:   100,
			AccessExpiresAt:  200,
			RefreshExpiresAt: 300,
			SID:              "session",
			JTI:              "token",
		},
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["status"] != string(model.SocialAuthAuthenticated) {
		t.Fatalf("typed status missing: %#v", response)
	}
	if response["session"] == nil {
		t.Fatalf("typed session missing: %#v", response)
	}
	for _, legacyField := range []string{"accessToken", "refreshToken", "usrSeq", "usrId", "usrName", "usrStatus", "sid", "jti"} {
		if _, exists := response[legacyField]; exists {
			t.Fatalf("legacy flat field %q must be absent: %#v", legacyField, response)
		}
	}
}

func TestKakaoLinkRequiredResponseUsesCanonicalContextOnly(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeMobileAuthResult(recorder, model.SocialAuthResult{
		Status: model.SocialAuthLinkRequired,
		LinkRequired: &model.SocialLinkContext{
			LinkToken: "link-token",
			Provider:  model.SocialProviderKakao,
			Profile: model.SocialProviderProfile{
				Email: "prefill@example.com",
			},
		},
	})

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["linkRequired"] == nil {
		t.Fatalf("typed link context missing: %#v", response)
	}
	for _, legacyField := range []string{"linkToken", "provider", "email", "nickname"} {
		if _, exists := response[legacyField]; exists {
			t.Fatalf("legacy flat field %q must be absent: %#v", legacyField, response)
		}
	}
}

func TestSocialRejectedResponseUsesCanonicalErrorEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeMobileAuthResult(recorder, model.SocialAuthResult{
		Status: model.SocialAuthRejected,
		Rejected: &model.SocialAuthRejection{
			Code:    "ACCOUNT_SUSPENDED",
			Message: "이 계정은 현재 로그인할 수 없습니다.",
		},
	})

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["code"] != "ACCOUNT_SUSPENDED" || response["message"] == nil {
		t.Fatalf("error envelope = %#v", response)
	}
	if _, exists := response["status"]; exists {
		t.Fatalf("social result status must not wrap an error: %#v", response)
	}
}
