package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

func TestAppleIdentityTokenValidation(t *testing.T) {
	signingKey := mustRSAKey(t)
	otherKey := mustRSAKey(t)
	jwksServer := appleJWKSServer(t, "apple-key", &signingKey.PublicKey)
	defer jwksServer.Close()

	verifier := &AppleIdentityVerifier{
		config: config.AppleConfig{
			AllowedAudiences: []string{"com.daeil.dflhsafv2.web", "com.daeil.dflhsafv2"},
		},
		jwks: newAppleJWKSCache(jwksServer.URL, time.Hour, jwksServer.Client()),
	}
	now := time.Now()
	validClaims := appleIdentityClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    appleIssuer,
			Subject:   "apple-subject",
			Audience:  jwt.ClaimStrings{"com.daeil.dflhsafv2"},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Nonce:         "expected-nonce-hash",
		Email:         "relay@privaterelay.appleid.com",
		EmailVerified: true,
	}

	tests := []struct {
		name          string
		claims        appleIdentityClaims
		key           *rsa.PrivateKey
		expectedNonce string
		wantError     bool
	}{
		{name: "valid private relay email", claims: validClaims, key: signingKey, expectedNonce: validClaims.Nonce},
		{name: "invalid signature", claims: validClaims, key: otherKey, expectedNonce: validClaims.Nonce, wantError: true},
		{name: "invalid nonce", claims: validClaims, key: signingKey, expectedNonce: "different", wantError: true},
		{name: "invalid issuer", claims: withAppleIssuer(validClaims, "https://attacker.example"), key: signingKey, expectedNonce: validClaims.Nonce, wantError: true},
		{name: "invalid audience", claims: withAppleAudience(validClaims, "other-client"), key: signingKey, expectedNonce: validClaims.Nonce, wantError: true},
		{name: "expired", claims: withAppleExpiry(validClaims, now.Add(-time.Minute)), key: signingKey, expectedNonce: validClaims.Nonce, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := signAppleClaims(t, test.claims, test.key, "apple-key")
			claims, err := verifier.verifyIdentityToken(context.Background(), raw, test.expectedNonce)
			if test.wantError {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if claims.Email != "relay@privaterelay.appleid.com" || !claims.emailIsVerified() {
				t.Fatalf("private relay email was not preserved: %#v", claims)
			}
		})
	}
}

func TestAppleServerNotificationSignatureAndEventValidation(t *testing.T) {
	signingKey := mustRSAKey(t)
	otherKey := mustRSAKey(t)
	jwksServer := appleJWKSServer(t, "apple-key", &signingKey.PublicKey)
	defer jwksServer.Close()
	verifier := &AppleIdentityVerifier{
		config: config.AppleConfig{
			AllowedAudiences: []string{"com.daeil.dflhsafv2.web", "com.daeil.dflhsafv2"},
		},
		jwks: newAppleJWKSCache(jwksServer.URL, time.Hour, jwksServer.Client()),
	}
	claims := appleNotificationClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   appleIssuer,
			Audience: jwt.ClaimStrings{"com.daeil.dflhsafv2"},
			IssuedAt: jwt.NewNumericDate(time.Now()),
			ID:       "notification-id",
		},
	}
	claims.Events.Type = "email-disabled"
	claims.Events.Subject = "apple-subject"
	claims.Events.Email = "relay@privaterelay.appleid.com"

	valid := signAppleNotification(t, claims, signingKey, "apple-key")
	notification, err := verifier.VerifyServerNotification(context.Background(), valid)
	if err != nil {
		t.Fatal(err)
	}
	if notification.Type != "email-disabled" || notification.Subject != "apple-subject" {
		t.Fatalf("notification = %#v", notification)
	}

	invalid := signAppleNotification(t, claims, otherKey, "apple-key")
	if _, err := verifier.VerifyServerNotification(context.Background(), invalid); err == nil {
		t.Fatal("invalid notification signature was accepted")
	}
}

func TestAppleJWKSCacheRefreshesWhenAppleRotatesToUnknownKey(t *testing.T) {
	oldKey := mustRSAKey(t)
	newKey := mustRSAKey(t)
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := requestCount.Add(1)
		keyID := "old-key"
		key := &oldKey.PublicKey
		if count > 1 {
			keyID = "new-key"
			key = &newKey.PublicKey
		}
		exponent := big.NewInt(int64(key.E)).Bytes()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": []map[string]string{{
				"kid": keyID,
				"kty": "RSA",
				"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(exponent),
			}},
		})
	}))
	defer server.Close()
	cache := newAppleJWKSCache(server.URL, time.Hour, server.Client())

	if _, err := cache.key(context.Background(), "old-key"); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.key(context.Background(), "new-key"); err != nil {
		t.Fatal(err)
	}
	if requestCount.Load() != 2 {
		t.Fatalf("JWKS request count = %d, want 2", requestCount.Load())
	}
}

func TestAppleVerifyExchangesCodeAndAllowsRelayEmailWithoutName(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	signingKey := mustRSAKey(t)
	jwksServer := appleJWKSServer(t, "apple-key", &signingKey.PublicKey)
	defer jwksServer.Close()
	clientSigningKey := mustECDSAKey(t)
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			t.Error(err)
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if request.Form.Get("code") != "single-use-code" ||
			request.Form.Get("grant_type") != "authorization_code" ||
			request.Form.Get("client_id") != "com.daeil.dflhsafv2" {
			t.Errorf("unexpected token exchange form")
		}
		clientSecret := request.Form.Get("client_secret")
		parsed, err := jwt.Parse(
			clientSecret,
			func(token *jwt.Token) (interface{}, error) {
				return &clientSigningKey.PublicKey, nil
			},
			jwt.WithIssuer("TEAMID1234"),
			jwt.WithAudience(appleIssuer),
			jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}),
		)
		if err != nil || !parsed.Valid {
			t.Errorf("invalid generated Apple client secret: %v", err)
		}
		if parsed.Header["kid"] != "APPLEKEY01" {
			t.Errorf("client secret kid = %v", parsed.Header["kid"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token":  "apple-access",
			"refresh_token": "apple-refresh",
			"id_token":      "apple-id",
		})
	}))
	defer tokenServer.Close()

	now := time.Now()
	expectedNonce := "expected-nonce-hash"
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT NONCE_HASH, EXPIRES_AT, CONSUMED_AT`).
		WithArgs("challenge").
		WillReturnRows(sqlmock.NewRows([]string{"NONCE_HASH", "EXPIRES_AT", "CONSUMED_AT"}).
			AddRow(expectedNonce, now.Add(time.Minute), nil))
	mock.ExpectExec(`UPDATE ALUMNI_APPLE_NONCE_CHALLENGE`).
		WithArgs("challenge").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec(`INSERT INTO ALUMNI_APPLE_CODE_REPLAY`).
		WithArgs(sha256Hex("single-use-code")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	cfg := config.AppleConfig{
		TeamID:           "TEAMID1234",
		KeyID:            "APPLEKEY01",
		ClientID:         "com.daeil.dflhsafv2",
		PrivateKey:       encodeECDSAPrivateKey(t, clientSigningKey),
		AllowedAudiences: []string{"com.daeil.dflhsafv2"},
		TokenURL:         tokenServer.URL,
		JWKSCacheTTL:     time.Hour,
	}
	verifier := NewAppleIdentityVerifier(auth, cfg)
	verifier.httpClient = tokenServer.Client()
	verifier.jwks = newAppleJWKSCache(jwksServer.URL, time.Hour, jwksServer.Client())
	identityToken := signAppleClaims(t, appleIdentityClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    appleIssuer,
			Subject:   "apple-subject",
			Audience:  jwt.ClaimStrings{"com.daeil.dflhsafv2"},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Nonce:         expectedNonce,
		Email:         "relay@privaterelay.appleid.com",
		EmailVerified: "true",
	}, signingKey, "apple-key")

	account, err := verifier.Verify(context.Background(), model.AppleAuthorization{
		ChallengeID:       "challenge",
		IdentityToken:     identityToken,
		AuthorizationCode: "single-use-code",
	})

	if err != nil {
		t.Fatal(err)
	}
	if account.Identity.Subject != "apple-subject" ||
		account.Identity.Email != "relay@privaterelay.appleid.com" ||
		!account.Identity.EmailVerified {
		t.Fatalf("identity = %#v", account.Identity)
	}
	if account.Profile.DisplayName != "" || account.Profile.GivenName != "" || account.Profile.FamilyName != "" {
		t.Fatalf("missing relogin name was not preserved: %#v", account.Profile)
	}
	if account.RevocationToken != "apple-refresh" {
		t.Fatal("Apple refresh token was not retained for revocation")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func signAppleClaims(t *testing.T, claims appleIdentityClaims, key *rsa.PrivateKey, kid string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func signAppleNotification(t *testing.T, claims appleNotificationClaims, key *rsa.PrivateKey, kid string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func appleJWKSServer(t *testing.T, kid string, key *rsa.PublicKey) *httptest.Server {
	t.Helper()
	exponent := big.NewInt(int64(key.E)).Bytes()
	payload := map[string]interface{}{
		"keys": []map[string]string{{
			"kid": kid,
			"kty": "RSA",
			"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(exponent),
		}},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Error(err)
		}
	}))
}

func mustRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func mustECDSAKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func encodeECDSAPrivateKey(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}))
}

func withAppleIssuer(claims appleIdentityClaims, issuer string) appleIdentityClaims {
	claims.Issuer = issuer
	return claims
}

func withAppleAudience(claims appleIdentityClaims, audience string) appleIdentityClaims {
	claims.Audience = jwt.ClaimStrings{audience}
	return claims
}

func withAppleExpiry(claims appleIdentityClaims, expiry time.Time) appleIdentityClaims {
	claims.ExpiresAt = jwt.NewNumericDate(expiry)
	return claims
}
