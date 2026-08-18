package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestKakaoCompatibilityResponseKeepsFlatSessionFields(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeKakaoCompatibilityResult(recorder, model.SocialAuthResult{
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
	if response["accessToken"] != "access" || response["refreshToken"] != "refresh" {
		t.Fatalf("legacy flat token fields missing: %#v", response)
	}
	if response["session"] == nil {
		t.Fatalf("typed session missing: %#v", response)
	}
}

func TestKakaoCompatibilityResponseExposesTypedLinkContext(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeKakaoCompatibilityResult(recorder, model.SocialAuthResult{
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
	if response["linkToken"] != "link-token" || response["linkRequired"] == nil {
		t.Fatalf("link compatibility fields missing: %#v", response)
	}
}
