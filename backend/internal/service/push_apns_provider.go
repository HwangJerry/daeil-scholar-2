package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

const (
	apnsAPIVersion       = "/3/device/"
	apnsDefaultHost      = "api.push.apple.com"
	apnsSandboxHost      = "api.sandbox.push.apple.com"
	apnsAuthHeaderPrefix = "Bearer "
	apnsTokenCacheMaxAge = 45 * time.Minute
)

type APNsResponseError struct {
	StatusCode int
	Reason     string
	Body       string
}

func (e *APNsResponseError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("apns status=%d reason=%s body=%s", e.StatusCode, e.Reason, e.Body)
	}
	return fmt.Sprintf("apns status=%d body=%s", e.StatusCode, e.Body)
}

func (e *APNsResponseError) InvalidDeviceToken() bool {
	switch e.Reason {
	case "BadDeviceToken", "Unregistered", "DeviceTokenNotForTopic":
		return true
	default:
		return false
	}
}

type APNsPushProvider struct {
	cfg        config.PushConfig
	logger     zerolog.Logger
	httpClient *http.Client

	mu         sync.Mutex
	cachedJWT  string
	cachedAt   time.Time
	privateKey interface{}
}

func NewAPNsPushProvider(cfg config.PushConfig, logger zerolog.Logger) *APNsPushProvider {
	return &APNsPushProvider{
		cfg:        cfg,
		logger:     logger,
		httpClient: &http.Client{Timeout: cfg.APNsRequestTimeout},
	}
}

func (p *APNsPushProvider) SendPush(ctx context.Context, deviceToken string, title string, body string, data map[string]any) error {
	deviceToken = strings.TrimSpace(deviceToken)
	if deviceToken == "" {
		return nil
	}

	key, err := p.loadSigningKey()
	if err != nil {
		p.logger.Error().Err(err).Msg("push: APNs signing key load failed; notification skipped")
		return err
	}
	signed, err := p.signedToken(key)
	if err != nil {
		return err
	}

	apnsPayload := map[string]any{
		"aps": map[string]any{
			"alert": map[string]any{
				"title": title,
				"body":  body,
			},
			"sound": "default",
		},
	}
	for k, v := range data {
		apnsPayload[k] = v
	}

	bodyData, err := json.Marshal(apnsPayload)
	if err != nil {
		return err
	}

	u := &url.URL{
		Scheme: "https",
		Host:   "api.push.apple.com",
		Path:   apnsAPIVersion + deviceToken,
	}
	if p.cfg.APNsUseSandbox {
		u.Host = apnsSandboxHost
	} else {
		u.Host = apnsDefaultHost
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(bodyData))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s%s", apnsAuthHeaderPrefix, signed))
	req.Header.Set("apns-topic", p.cfg.APNsBundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	respBodyText := strings.TrimSpace(string(respBody))
	return &APNsResponseError{
		StatusCode: resp.StatusCode,
		Reason:     parseAPNsReason(respBody),
		Body:       respBodyText,
	}
}

func parseAPNsReason(respBody []byte) string {
	var payload struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return ""
	}
	return payload.Reason
}

func (p *APNsPushProvider) loadSigningKey() (interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.privateKey != nil {
		return p.privateKey, nil
	}
	if p.cfg.APNsKeyID == "" || (p.cfg.APNsKeyPath == "" && p.cfg.APNsKeyValue == "") || p.cfg.APNSTeamID == "" || p.cfg.APNsBundleID == "" {
		return nil, fmt.Errorf("apns config is incomplete")
	}

	keyData := strings.TrimSpace(p.cfg.APNsKeyValue)
	if keyData == "" && p.cfg.APNsKeyPath != "" {
		raw, err := os.ReadFile(p.cfg.APNsKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read apns key file: %w", err)
		}
		keyData = string(raw)
	}
	if keyData == "" {
		return nil, fmt.Errorf("apns key not configured")
	}

	parsed, err := jwt.ParseECPrivateKeyFromPEM([]byte(keyData))
	if err != nil {
		return nil, err
	}
	p.privateKey = parsed
	return parsed, nil
}

func (p *APNsPushProvider) signedToken(privateKey interface{}) (string, error) {
	now := time.Now().Unix()
	p.mu.Lock()
	cachedToken := p.cachedJWT
	cachedFresh := cachedToken != "" && time.Since(p.cachedAt) < apnsTokenCacheMaxAge
	p.mu.Unlock()
	if cachedFresh {
		return cachedToken, nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": p.cfg.APNSTeamID,
		"iat": now,
		"exp": now + 60*60,
	})
	token.Header["alg"] = "ES256"
	token.Header["kid"] = p.cfg.APNsKeyID

	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}

	p.mu.Lock()
	p.cachedJWT = signed
	p.cachedAt = time.Now()
	p.mu.Unlock()

	return signed, nil
}
