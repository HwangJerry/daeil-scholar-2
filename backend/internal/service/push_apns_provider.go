package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
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

type APNsConfigurationError struct {
	Environment string
}

func (e *APNsConfigurationError) Error() string {
	return fmt.Sprintf("apns credentials are not configured for environment %q", e.Environment)
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
	authTokens map[string]*token.Token
}

func NewAPNsPushProvider(cfg config.PushConfig, logger zerolog.Logger) (*APNsPushProvider, error) {
	if cfg.APNSTeamID == "" {
		return nil, fmt.Errorf("apns team id is required when APNs credentials are configured")
	}
	if cfg.APNsBundleID == "" {
		return nil, fmt.Errorf("apns bundle id is required when APNs credentials are configured")
	}

	authTokens := make(map[string]*token.Token, 2)
	configuredEnvironments := make([]string, 0, 2)
	credentials := []struct {
		environment string
		credential  config.APNsCredentialConfig
	}{
		{environment: "sandbox", credential: cfg.APNsSandbox},
		{environment: "production", credential: cfg.APNsProduction},
	}
	for _, item := range credentials {
		if !item.credential.Configured() {
			continue
		}
		authToken, err := loadAPNsAuthToken(item.environment, cfg.APNSTeamID, item.credential)
		if err != nil {
			return nil, err
		}
		authTokens[item.environment] = authToken
		configuredEnvironments = append(configuredEnvironments, item.environment)
	}
	if len(authTokens) == 0 {
		return nil, fmt.Errorf("apns credentials are not configured")
	}

	provider := &APNsPushProvider{
		cfg:        cfg,
		logger:     logger,
		httpClient: &http.Client{Timeout: cfg.APNsRequestTimeout},
		authTokens: authTokens,
	}
	logger.Info().Strs("environments", configuredEnvironments).Msg("push: APNs provider configured")
	return provider, nil
}

func (p *APNsPushProvider) SendPush(ctx context.Context, notification PushNotification) error {
	notification.DeviceToken = strings.TrimSpace(notification.DeviceToken)
	if notification.DeviceToken == "" {
		return nil
	}

	environment := apnsEnvironmentForNotification(notification.APNsEnvironment, p.cfg.APNsEnvironment)
	authToken, ok := p.authTokens[environment]
	if !ok {
		return &APNsConfigurationError{Environment: environment}
	}

	bodyData, err := buildAPNsPayload(notification)
	if err != nil {
		return err
	}

	apnsNotification := buildAPNsNotification(p.cfg, notification, bodyData)

	client := apns2.NewTokenClient(authToken)
	client.HTTPClient = p.httpClient
	if environment == "sandbox" {
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
	env := apnsEnvironmentForNotification(tokenEnvironment, defaultEnvironment)
	if env == "sandbox" {
		return apnsSandboxHost
	}
	return apnsProductionHost
}

func apnsEnvironmentForNotification(tokenEnvironment string, defaultEnvironment string) string {
	environment := normalizeAPNsEnvironment(tokenEnvironment)
	if environment == "" {
		environment = normalizeAPNsEnvironment(defaultEnvironment)
	}
	if environment == "" {
		return "production"
	}
	return environment
}

func loadAPNsAuthToken(environment string, teamID string, credential config.APNsCredentialConfig) (*token.Token, error) {
	if credential.KeyID == "" {
		return nil, fmt.Errorf("apns %s key id is required", environment)
	}
	if credential.KeyPath == "" && credential.KeyValue == "" {
		return nil, fmt.Errorf("apns %s private key is required", environment)
	}

	keyData := strings.TrimSpace(credential.KeyValue)
	keyData = strings.ReplaceAll(keyData, `\n`, "\n")
	if keyData == "" && credential.KeyPath != "" {
		raw, err := os.ReadFile(credential.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("read apns %s key file: %w", environment, err)
		}
		keyData = string(raw)
	}
	if keyData == "" {
		return nil, fmt.Errorf("apns %s private key is empty", environment)
	}

	authKey, err := token.AuthKeyFromBytes([]byte(keyData))
	if err != nil {
		return nil, fmt.Errorf("parse apns %s private key: %w", environment, err)
	}
	return &token.Token{
		AuthKey: authKey,
		KeyID:   credential.KeyID,
		TeamID:  teamID,
	}, nil
}
