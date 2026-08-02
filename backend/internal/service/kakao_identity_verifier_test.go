package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestKakaoIdentityVerifierValidAndInvalidAccessToken(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantError  bool
	}{
		{
			name:       "valid",
			statusCode: http.StatusOK,
			body:       `{"id":12345,"kakao_account":{"email":"member@example.com","profile":{"nickname":"Member"}}}`,
		},
		{name: "invalid", statusCode: http.StatusUnauthorized, body: `{"msg":"invalid token"}`, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			auth := &AuthService{
				cfg:    &config.Config{JWT: config.JWTConfig{MaxAge: time.Hour}},
				cache:  cache.New(time.Minute, time.Minute),
				logger: zerolog.Nop(),
				httpClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					if request.Header.Get("Authorization") != "Bearer opaque-token" {
						t.Fatalf("authorization header = %q", request.Header.Get("Authorization"))
					}
					return &http.Response{
						StatusCode: test.statusCode,
						Body:       io.NopCloser(strings.NewReader(test.body)),
						Header:     make(http.Header),
					}, nil
				})},
			}
			account, err := NewKakaoIdentityVerifier(auth).Verify(context.Background(), model.KakaoAuthorization{
				AccessToken: "opaque-token",
			})
			if test.wantError {
				if err == nil {
					t.Fatal("expected verifier error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if account.Identity.Subject != "12345" || account.Identity.Email != "member@example.com" {
				t.Fatalf("unexpected identity: %#v", account.Identity)
			}
		})
	}
}
