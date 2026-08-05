package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestMobileAuthResponseUsesCanonicalAuthenticatedEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeMobileAuthResult(recorder, model.SocialAuthResult{
		Status: model.SocialAuthAuthenticated,
		Session: &model.MobileSession{
			User: model.AuthUser{
				USRSeq:    42,
				USRID:     "member",
				USRName:   "Member",
				Email:     "member@example.com",
				AdminRole: nil,
				Verification: model.AlumniVerification{
					Status: model.VerificationApproved,
				},
			},
			AccessToken:      "fixture-access",
			RefreshToken:     "fixture-refresh",
			AccessIssuedAt:   100,
			AccessExpiresAt:  200,
			RefreshExpiresAt: 300,
			SID:              "fixture-session",
			JTI:              "fixture-jti",
		},
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["status"] != "authenticated" {
		t.Fatalf("status = %#v", response["status"])
	}
	if response["session"] == nil {
		t.Fatalf("session missing: %#v", response)
	}
	for _, legacyField := range []string{"accessToken", "refreshToken", "usrSeq", "usrId", "usrName", "usrStatus", "sid", "jti"} {
		if _, exists := response[legacyField]; exists {
			t.Fatalf("legacy flat field %q must be absent: %#v", legacyField, response)
		}
	}
}

func TestMobileAuthResponseUsesCanonicalLinkRequiredEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeMobileAuthResult(recorder, model.SocialAuthResult{
		Status: model.SocialAuthLinkRequired,
		LinkRequired: &model.SocialLinkContext{
			LinkToken: "fixture-link-token",
			Provider:  model.SocialProviderKakao,
			ExpiresAt: 300,
			Profile: model.SocialProviderProfile{
				DisplayName: "Member",
				Email:       "member@example.com",
			},
		},
	})

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["status"] != "linkRequired" || response["linkRequired"] == nil {
		t.Fatalf("canonical linkRequired missing: %#v", response)
	}
	linkRequired, ok := response["linkRequired"].(map[string]any)
	if !ok || linkRequired["provider"] != "KT" {
		t.Fatalf("canonical provider = %#v", linkRequired["provider"])
	}
	for _, legacyField := range []string{"linkToken", "provider", "socialId", "email", "nickname", "profileImageUrl"} {
		if _, exists := response[legacyField]; exists {
			t.Fatalf("legacy flat field %q must be absent: %#v", legacyField, response)
		}
	}
}

func TestMobileAuthResponseRejectsNonCanonicalTopLevelStatuses(t *testing.T) {
	for _, status := range []model.SocialAuthStatus{"pending", "rejected"} {
		t.Run(string(status), func(t *testing.T) {
			recorder := httptest.NewRecorder()
			writeMobileAuthResult(recorder, model.SocialAuthResult{Status: status})

			if recorder.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
			}
			var response map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if _, exists := response["status"]; exists {
				t.Fatalf("non-canonical top-level status leaked: %#v", response)
			}
		})
	}
}
