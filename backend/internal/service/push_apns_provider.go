package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/rs/zerolog"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
)

const (
	apnsProductionHost = "api.push.apple.com"
	apnsSandboxHost    = "api.sandbox.push.apple.com"
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

	mu        sync.Mutex
	authToken *token.Token
}

func NewAPNsPushProvider(cfg config.PushConfig, logger zerolog.Logger) *APNsPushProvider {
	return &APNsPushProvider{
		cfg:        cfg,
		logger:     logger,
		httpClient: &http.Client{Timeout: cfg.APNsRequestTimeout},
	}
}

func (p *APNsPushProvider) SendPush(ctx context.Context, notification PushNotification) error {
	notification.DeviceToken = strings.TrimSpace(notification.DeviceToken)
	if notification.DeviceToken == "" {
		return nil
	}

	authToken, err := p.loadAuthToken()
	if err != nil {
		p.logger.Error().Err(err).Msg("push: APNs signing key load failed; notification skipped")
		return err
	}

	bodyData, err := buildAPNsPayload(notification)
	if err != nil {
		return err
	}

	apnsNotification := buildAPNsNotification(p.cfg, notification, bodyData)

	client := apns2.NewTokenClient(authToken)
	client.HTTPClient = p.httpClient
	if apnsHostForEnvironment(notification.APNsEnvironment, p.cfg.APNsEnvironment) == apnsSandboxHost {
		client.Development()
	} else {
		client.Production()
	}
	resp, err := client.PushWithContext(ctx, apnsNotification)
	if err != nil {
		return err
	}

	if resp.Sent() {
		return nil
	}

	return &APNsResponseError{
		StatusCode: resp.StatusCode,
		Reason:     resp.Reason,
	}
}

func buildAPNsPayload(notification PushNotification) ([]byte, error) {
	apnsPayload := map[string]any{
		"aps": map[string]any{
			"alert": map[string]any{
				"title": notification.Title,
				"body":  notification.Body,
			},
			"sound": "default",
		},
	}
	for k, v := range notification.Payload.CustomPayload() {
		apnsPayload[k] = v
	}
	return json.Marshal(apnsPayload)
}

func buildAPNsNotification(cfg config.PushConfig, notification PushNotification, payloadBytes []byte) *apns2.Notification {
	apnsNotification := &apns2.Notification{
		DeviceToken: notification.DeviceToken,
		Topic:       apnsTopicForNotification(cfg, notification),
		PushType:    apns2.PushTypeAlert,
		Priority:    apns2.PriorityHigh,
		Payload:     payloadBytes,
	}
	if notification.Payload.TTLSec > 0 {
		apnsNotification.Expiration = notification.Payload.SentAt.UTC().Add(time.Duration(notification.Payload.TTLSec) * time.Second)
	}
	if notification.Payload.CollapseKey != "" {
		apnsNotification.CollapseID = notification.Payload.CollapseKey
	}
	return apnsNotification
}

func makeAPNsHeaders(cfg config.PushConfig, notification PushNotification) http.Header {
	headers := http.Header{}
	headers.Set("apns-topic", apnsTopicForNotification(cfg, notification))
	headers.Set("apns-push-type", "alert")
	headers.Set("apns-priority", "10")
	if notification.Payload.TTLSec > 0 {
		expiration := notification.Payload.SentAt.UTC().Add(time.Duration(notification.Payload.TTLSec) * time.Second).Unix()
		headers.Set("apns-expiration", fmt.Sprintf("%d", expiration))
	}
	if notification.Payload.CollapseKey != "" {
		headers.Set("apns-collapse-id", notification.Payload.CollapseKey)
	}
	return headers
}

func apnsTopicForNotification(cfg config.PushConfig, notification PushNotification) string {
	topic := strings.TrimSpace(notification.BundleID)
	if topic == "" {
		topic = cfg.APNsBundleID
	}
	return topic
}

func apnsHostForEnvironment(tokenEnvironment string, defaultEnvironment string) string {
	env := normalizeAPNsEnvironment(tokenEnvironment)
	if env == "" {
		env = normalizeAPNsEnvironment(defaultEnvironment)
	}
	if env == "sandbox" {
		return apnsSandboxHost
	}
	return apnsProductionHost
}

func (p *APNsPushProvider) loadAuthToken() (*token.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.authToken != nil {
		return p.authToken, nil
	}
	if p.cfg.APNsKeyID == "" || (p.cfg.APNsKeyPath == "" && p.cfg.APNsKeyValue == "") || p.cfg.APNSTeamID == "" || p.cfg.APNsBundleID == "" {
		return nil, fmt.Errorf("apns config is incomplete")
	}

	keyData := strings.TrimSpace(p.cfg.APNsKeyValue)
	keyData = strings.ReplaceAll(keyData, `\n`, "\n")
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

	authKey, err := token.AuthKeyFromBytes([]byte(keyData))
	if err != nil {
		return nil, err
	}
	p.authToken = &token.Token{
		AuthKey: authKey,
		KeyID:   p.cfg.APNsKeyID,
		TeamID:  p.cfg.APNSTeamID,
	}
	return p.authToken, nil
}
