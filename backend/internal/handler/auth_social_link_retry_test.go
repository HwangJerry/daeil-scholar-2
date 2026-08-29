package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/patrickmn/go-cache"
)

func TestSocialLinkValidationFailureDoesNotConsumeToken(t *testing.T) {
	tokenStore := service.NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	if _, err := tokenStore.Put("retry-token", model.SocialLinkData{}, time.Minute); err != nil {
		t.Fatal(err)
	}
	handler := &AuthHandler{socialLinkTokens: tokenStore}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/social/link",
		strings.NewReader(`{
			"token":"retry-token",
			"mode":"new",
			"name":"Member",
			"email":"relay@privaterelay.appleid.com",
			"phone":"010-1234-5678",
			"fn":"not-a-number",
			"fmDept":"영어"
		}`),
	)
	response := httptest.NewRecorder()

	handler.SocialLink(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if _, err := tokenStore.Begin("retry-token"); err != nil {
		t.Fatalf("validation failure consumed token: %v", err)
	}
}

func TestSocialLinkRejectsUnsupportedModeWithoutConsumingToken(t *testing.T) {
	tokenStore := service.NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	if _, err := tokenStore.Put("unsupported-mode-token", model.SocialLinkData{}, time.Minute); err != nil {
		t.Fatal(err)
	}
	handler := &AuthHandler{socialLinkTokens: tokenStore}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/social/link",
		strings.NewReader(`{"token":"unsupported-mode-token","mode":"unsupported"}`),
	)
	response := httptest.NewRecorder()

	handler.SocialLink(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"UNSUPPORTED_SOCIAL_LINK_MODE"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
	if _, err := tokenStore.Begin("unsupported-mode-token"); err != nil {
		t.Fatalf("unsupported mode consumed token: %v", err)
	}
}
