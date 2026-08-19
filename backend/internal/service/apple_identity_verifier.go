package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

const appleIssuer = "https://appleid.apple.com"

type AppleIdentityVerifier struct {
	auth       *AuthService
	config     config.AppleConfig
	httpClient *http.Client
	jwks       *appleJWKSCache
	now        func() time.Time
}

func NewAppleIdentityVerifier(auth *AuthService, cfg config.AppleConfig) *AppleIdentityVerifier {
	client := &http.Client{Timeout: 10 * time.Second}
	return &AppleIdentityVerifier{
		auth:       auth,
		config:     cfg,
		httpClient: client,
		jwks:       newAppleJWKSCache(cfg.JWKSURL, cfg.JWKSCacheTTL, client),
		now:        time.Now,
	}
}

func (*AppleIdentityVerifier) Provider() model.SocialProvider {
	return model.SocialProviderApple
}

func (v *AppleIdentityVerifier) CreateChallenge() (model.AppleChallenge, error) {
	challengeID, err := randomBase64URL(32)
	if err != nil {
		return model.AppleChallenge{}, err
	}
	nonce, err := randomBase64URL(32)
	if err != nil {
		return model.AppleChallenge{}, err
	}
	nonceHash := sha256Hex(nonce)
	expiresAt := v.now().Add(v.config.ChallengeTTL)
	if err := v.auth.repo.InsertAppleChallenge(challengeID, nonceHash, expiresAt); err != nil {
		return model.AppleChallenge{}, err
	}
	return model.AppleChallenge{
		ID:        challengeID,
		Nonce:     nonce,
		NonceHash: nonceHash,
		ExpiresAt: expiresAt,
	}, nil
}

func (v *AppleIdentityVerifier) Verify(ctx context.Context, authorization model.SocialAuthorization) (VerifiedSocialAccount, error) {
	apple, ok := authorization.(model.AppleAuthorization)
	if !ok {
		return VerifiedSocialAccount{}, errors.New("missing apple authorization")
	}
	if strings.TrimSpace(apple.ChallengeID) == "" ||
		strings.TrimSpace(apple.IdentityToken) == "" ||
		strings.TrimSpace(apple.AuthorizationCode) == "" {
		return VerifiedSocialAccount{}, errors.New("incomplete apple authorization")
	}

	expectedNonce, err := v.auth.repo.ConsumeAppleChallenge(apple.ChallengeID)
	if err != nil {
		return VerifiedSocialAccount{}, err
	}
	claims, err := v.verifyIdentityToken(ctx, apple.IdentityToken, expectedNonce)
	if err != nil {
		return VerifiedSocialAccount{}, err
	}

	codeHash := sha256Hex(apple.AuthorizationCode)
	if err := v.auth.repo.ConsumeAppleAuthorizationCode(codeHash); err != nil {
		return VerifiedSocialAccount{}, err
	}
	tokenResponse, err := v.exchangeAuthorizationCode(ctx, apple.AuthorizationCode)
	if err != nil {
		return VerifiedSocialAccount{}, err
	}

	displayName := strings.TrimSpace(strings.Join([]string{apple.GivenName, apple.FamilyName}, " "))
	return VerifiedSocialAccount{
		Identity: model.VerifiedSocialIdentity{
			Provider:      model.SocialProviderApple,
			Subject:       claims.Subject,
			Email:         claims.Email,
			EmailVerified: claims.emailIsVerified(),
		},
		Profile: model.SocialProviderProfile{
			DisplayName: displayName,
			GivenName:   apple.GivenName,
			FamilyName:  apple.FamilyName,
			Email:       claims.Email,
		},
		RevocationToken: tokenResponse.RefreshToken,
	}, nil
}

func (v *AppleIdentityVerifier) verifyIdentityToken(ctx context.Context, rawToken string, expectedNonce string) (*appleIdentityClaims, error) {
	claims := &appleIdentityClaims{}
	options := []jwt.ParserOption{
		jwt.WithIssuer(appleIssuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	}
	if len(v.config.AllowedAudiences) == 0 {
		return nil, errors.New("APPLE_ALLOWED_AUDIENCES is required")
	}
	parsed, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (interface{}, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("apple token missing kid")
		}
		return v.jwks.key(ctx, kid)
	}, options...)
	if err != nil || !parsed.Valid {
		return nil, fmt.Errorf("invalid apple identity token: %w", err)
	}
	if !containsAudience(claims.Audience, v.config.AllowedAudiences) {
		return nil, errors.New("invalid apple audience")
	}
	if subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(expectedNonce)) != 1 {
		return nil, errors.New("invalid apple nonce")
	}
	if claims.Subject == "" {
		return nil, errors.New("missing apple subject")
	}
	return claims, nil
}

type AppleServerNotification struct {
	Type    string
	Subject string
	Email   string
}

func (v *AppleIdentityVerifier) VerifyServerNotification(ctx context.Context, signedPayload string) (AppleServerNotification, error) {
	claims := &appleNotificationClaims{}
	if len(v.config.AllowedAudiences) == 0 {
		return AppleServerNotification{}, errors.New("APPLE_ALLOWED_AUDIENCES is required")
	}
	parsed, err := jwt.ParseWithClaims(
		signedPayload,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			kid, _ := token.Header["kid"].(string)
			if kid == "" {
				return nil, errors.New("apple notification missing kid")
			}
			return v.jwks.key(ctx, kid)
		},
		jwt.WithIssuer(appleIssuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuedAt(),
	)
	if err != nil || !parsed.Valid {
		return AppleServerNotification{}, fmt.Errorf("invalid apple notification: %w", err)
	}
	if !containsAudience(claims.Audience, v.config.AllowedAudiences) {
		return AppleServerNotification{}, errors.New("invalid apple notification audience")
	}
	if claims.ID == "" || claims.IssuedAt == nil {
		return AppleServerNotification{}, errors.New("apple notification metadata is incomplete")
	}
	if claims.Events.Subject == "" || claims.Events.Type == "" {
		return AppleServerNotification{}, errors.New("apple notification event is incomplete")
	}
	return AppleServerNotification{
		Type:    claims.Events.Type,
		Subject: claims.Events.Subject,
		Email:   claims.Events.Email,
	}, nil
}

type appleNotificationClaims struct {
	jwt.RegisteredClaims
	Events struct {
		Type    string `json:"type"`
		Subject string `json:"sub"`
		Email   string `json:"email"`
	} `json:"events"`
}

type appleIdentityClaims struct {
	jwt.RegisteredClaims
	Nonce         string      `json:"nonce"`
	Email         string      `json:"email"`
	EmailVerified interface{} `json:"email_verified"`
}

func (c appleIdentityClaims) emailIsVerified() bool {
	switch value := c.EmailVerified.(type) {
	case bool:
		return value
	case string:
		verified, _ := strconv.ParseBool(value)
		return verified
	default:
		return false
	}
}

type appleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

func (v *AppleIdentityVerifier) exchangeAuthorizationCode(ctx context.Context, code string) (appleTokenResponse, error) {
	clientSecret, err := v.clientSecret()
	if err != nil {
		return appleTokenResponse{}, err
	}
	form := url.Values{
		"client_id":     {v.config.ClientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, v.config.TokenURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return appleTokenResponse{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := v.httpClient.Do(request)
	if err != nil {
		return appleTokenResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, response.Body)
		return appleTokenResponse{}, fmt.Errorf("apple code exchange failed with status %d", response.StatusCode)
	}
	var payload appleTokenResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return appleTokenResponse{}, err
	}
	return payload, nil
}

func (v *AppleIdentityVerifier) RevokeToken(ctx context.Context, token string) error {
	clientSecret, err := v.clientSecret()
	if err != nil {
		return err
	}
	form := url.Values{
		"client_id":       {v.config.ClientID},
		"client_secret":   {clientSecret},
		"token":           {token},
		"token_type_hint": {"refresh_token"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, v.config.RevokeURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := v.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("apple revoke failed with status %d", response.StatusCode)
	}
	return nil
}

func (v *AppleIdentityVerifier) clientSecret() (string, error) {
	if v.config.TeamID == "" || v.config.KeyID == "" || v.config.ClientID == "" {
		return "", errors.New("apple client credentials are not configured")
	}
	privateKey, err := v.loadPrivateKey()
	if err != nil {
		return "", err
	}
	now := v.now()
	claims := jwt.RegisteredClaims{
		Issuer:    v.config.TeamID,
		Subject:   v.config.ClientID,
		Audience:  jwt.ClaimStrings{appleIssuer},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = v.config.KeyID
	return token.SignedString(privateKey)
}

func (v *AppleIdentityVerifier) loadPrivateKey() (*ecdsa.PrivateKey, error) {
	raw := strings.ReplaceAll(v.config.PrivateKey, `\n`, "\n")
	if raw == "" && v.config.PrivateKeyPath != "" {
		data, err := os.ReadFile(v.config.PrivateKeyPath)
		if err != nil {
			return nil, err
		}
		raw = string(data)
	}
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, errors.New("invalid apple private key PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("apple private key is not EC")
	}
	return ecdsaKey, nil
}

type appleJWKSCache struct {
	url        string
	ttl        time.Duration
	httpClient *http.Client
	mu         sync.Mutex
	keys       map[string]*rsa.PublicKey
	expiresAt  time.Time
}

func newAppleJWKSCache(url string, ttl time.Duration, client *http.Client) *appleJWKSCache {
	return &appleJWKSCache{
		url:        url,
		ttl:        ttl,
		httpClient: client,
		keys:       make(map[string]*rsa.PublicKey),
	}
}

func (c *appleJWKSCache) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if key := c.keys[kid]; key != nil && time.Now().Before(c.expiresAt) {
		return key, nil
	}
	if err := c.refresh(ctx); err != nil {
		return nil, err
	}
	key := c.keys[kid]
	if key == nil {
		return nil, errors.New("apple signing key not found")
	}
	return key, nil
}

func (c *appleJWKSCache) refresh(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("apple JWKS request failed with status %d", response.StatusCode)
	}
	var payload struct {
		Keys []struct {
			KID string `json:"kid"`
			KTY string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return err
	}
	keys := make(map[string]*rsa.PublicKey, len(payload.Keys))
	for _, jwk := range payload.Keys {
		if jwk.KTY != "RSA" || jwk.KID == "" {
			continue
		}
		key, err := rsaPublicKey(jwk.N, jwk.E)
		if err != nil {
			continue
		}
		keys[jwk.KID] = key
	}
	if len(keys) == 0 {
		return errors.New("apple JWKS contained no usable RSA keys")
	}
	c.keys = keys
	c.expiresAt = time.Now().Add(c.ttl)
	return nil
}

func rsaPublicKey(modulus string, exponent string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil {
		return nil, err
	}
	e := 0
	for _, value := range eBytes {
		e = e<<8 + int(value)
	}
	if e == 0 {
		return nil, errors.New("invalid RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

func randomBase64URL(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func containsAudience(actual jwt.ClaimStrings, allowed []string) bool {
	for _, current := range actual {
		for _, expected := range allowed {
			if current == expected {
				return true
			}
		}
	}
	return false
}
