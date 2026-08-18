package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
)

func TestLoginRateLimiterSeparatesSecuritySensitiveEndpointBuckets(t *testing.T) {
	cacheStore := cache.New(time.Minute, time.Minute)
	handler := LoginRateLimiter(cacheStore)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for range loginMaxAttempts {
		challengeRequest := httptest.NewRequest(http.MethodPost, "/api/auth/apple/challenge", nil)
		challengeRequest.RemoteAddr = "192.0.2.10:1234"
		challengeResponse := httptest.NewRecorder()
		handler.ServeHTTP(challengeResponse, challengeRequest)
		if challengeResponse.Code != http.StatusNoContent {
			t.Fatalf("challenge request status = %d, want %d", challengeResponse.Code, http.StatusNoContent)
		}
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/apple/mobile", nil)
	loginRequest.RemoteAddr = "192.0.2.10:1234"
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusNoContent {
		t.Fatalf("separate endpoint bucket status = %d, want %d", loginResponse.Code, http.StatusNoContent)
	}

	limitedRequest := httptest.NewRequest(http.MethodPost, "/api/auth/apple/challenge", nil)
	limitedRequest.RemoteAddr = "192.0.2.10:1234"
	limitedResponse := httptest.NewRecorder()
	handler.ServeHTTP(limitedResponse, limitedRequest)
	if limitedResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("rate-limited endpoint status = %d, want %d", limitedResponse.Code, http.StatusTooManyRequests)
	}
}
