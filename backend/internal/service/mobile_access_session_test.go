package service

import (
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

func TestMobileAccessTokenRestoresSessionIDForScopedLogout(t *testing.T) {
	svc := NewAuthService(nil, nil, &config.Config{JWT: config.JWTConfig{
		Secret: "fixture-secret",
		MaxAge: time.Hour,
	}}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())

	token, err := svc.GenerateMobileJWT(&model.AuthUser{USRSeq: 42, USRName: "Member", USRStatus: "CCC"}, "device-session")
	if err != nil {
		t.Fatal(err)
	}
	user, err := svc.ValidateMobileAccessToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if user.SessionID != "device-session" {
		t.Fatalf("session id = %q", user.SessionID)
	}
}
