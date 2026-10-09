package main

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"net/http"
	"testing"
	"time"
)

func TestSocialSignupCancelAPI(t *testing.T) {
	s := newGoldenServer(t)
	store := service.NewSocialLinkTokenStore(s.deps.cacheStore)
	_, _ = store.Put("cancel-api", model.SocialLinkData{AccessToken: "synthetic-secret", Email: "synthetic@example.test"}, time.Minute)
	for range 2 {
		s.request(t, http.MethodPost, "/api/auth/social/link/cancel", map[string]string{"token": "cancel-api"}, "", "ios", "100", 204)
	}
	s.request(t, http.MethodGet, "/api/auth/social/link/prefill?token=cancel-api", nil, "", "ios", "100", 404)
	body := s.request(t, http.MethodPost, "/api/auth/social/link", map[string]any{
		"token": "cancel-api", "name": "Synthetic", "email": "synthetic@example.test", "phone": goldenPhone, "fn": "20", "fmDept": "영어",
	}, "", "ios", "100", 400)
	assertGoldenError(t, body, "INVALID_TOKEN")
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_PHONE=?`, goldenPhone)
	s.request(t, http.MethodPost, "/api/auth/social/link/cancel", map[string]string{"token": "missing"}, "", "ios", "100", 204)
	_, _ = store.Put("expired", model.SocialLinkData{AccessToken: "synthetic"}, time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	s.request(t, http.MethodPost, "/api/auth/social/link/cancel", map[string]string{"token": "expired"}, "", "ios", "100", 204)
	if _, err := store.Begin("cancel-api"); err == nil {
		t.Fatal("cancelled API token can create member")
	}
	_, _ = store.Put("processing-api", model.SocialLinkData{}, time.Minute)
	lease, _ := store.Begin("processing-api")
	body = s.request(t, http.MethodPost, "/api/auth/social/link/cancel", map[string]string{"token": "processing-api"}, "", "ios", "100", 409)
	assertGoldenError(t, body, "TOKEN_IN_PROGRESS")
	_ = store.Consume(lease)
	body = s.request(t, http.MethodPost, "/api/auth/social/link/cancel", map[string]string{"token": "processing-api"}, "", "ios", "100", 409)
	assertGoldenError(t, body, "SIGNUP_ALREADY_COMPLETED")
}
