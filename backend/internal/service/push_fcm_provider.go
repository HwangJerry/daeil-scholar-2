package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/rs/zerolog"
	"google.golang.org/api/option"
)

type fcmMessagingClient interface {
	Send(ctx context.Context, message *messaging.Message) (string, error)
}

type FCMResponseError struct {
	Err                error
	Reason             string
	invalidDeviceToken bool
}

func (e *FCMResponseError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("fcm reason=%s error=%v", e.Reason, e.Err)
	}
	return fmt.Sprintf("fcm error=%v", e.Err)
}

func (e *FCMResponseError) Unwrap() error {
	return e.Err
}

func (e *FCMResponseError) InvalidDeviceToken() bool {
	return e.invalidDeviceToken
}

type FCMPushProvider struct {
	client         fcmMessagingClient
	logger         zerolog.Logger
	requestTimeout time.Duration
}

func NewFCMPushProvider(ctx context.Context, cfg config.PushConfig, logger zerolog.Logger) (*FCMPushProvider, error) {
	opts, err := fcmClientOptions(cfg)
	if err != nil {
		return nil, err
	}

	appCfg := &firebase.Config{}
	if strings.TrimSpace(cfg.FCMProjectID) != "" {
		appCfg.ProjectID = strings.TrimSpace(cfg.FCMProjectID)
	}

	app, err := firebase.NewApp(ctx, appCfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase app: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase messaging client: %w", err)
	}
	return &FCMPushProvider{
		client:         client,
		logger:         logger,
		requestTimeout: normalizePushProviderRequestTimeout(cfg.FCMRequestTimeout),
	}, nil
}

func NewFCMPushProviderForClient(client fcmMessagingClient, timeout time.Duration, logger zerolog.Logger) *FCMPushProvider {
	return &FCMPushProvider{
		client:         client,
		logger:         logger,
		requestTimeout: normalizePushProviderRequestTimeout(timeout),
	}
}

func (p *FCMPushProvider) SendPush(ctx context.Context, notification PushNotification) error {
	notification.DeviceToken = strings.TrimSpace(notification.DeviceToken)
	if notification.DeviceToken == "" {
		return nil
	}
	if p.client == nil {
		return fmt.Errorf("fcm messaging client is not configured")
	}

	data, err := fcmDataPayload(notification.Payload)
	if err != nil {
		return err
	}

	ttl := time.Duration(notification.Payload.TTLSec) * time.Second
	message := &messaging.Message{
		Token: notification.DeviceToken,
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Body,
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority:    "high",
			CollapseKey: notification.Payload.CollapseKey,
			TTL:         &ttl,
		},
	}

	sendCtx := ctx
	cancel := func() {}
	if p.requestTimeout > 0 {
		sendCtx, cancel = context.WithTimeout(ctx, p.requestTimeout)
	}
	defer cancel()

	if _, err := p.client.Send(sendCtx, message); err != nil {
		return newFCMResponseError(err)
	}
	return nil
}

func fcmClientOptions(cfg config.PushConfig) ([]option.ClientOption, error) {
	credentialsJSON := strings.TrimSpace(cfg.FCMCredentialsJSON)
	if credentialsJSON != "" {
		return []option.ClientOption{option.WithCredentialsJSON([]byte(credentialsJSON))}, nil
	}

	credentialsFile := strings.TrimSpace(cfg.FCMCredentialsFile)
	if credentialsFile != "" {
		return []option.ClientOption{option.WithCredentialsFile(credentialsFile)}, nil
	}

	return nil, fmt.Errorf("fcm config is incomplete: FCM_CREDENTIALS_JSON or FCM_CREDENTIALS_FILE is required")
}

func fcmDataPayload(payload PushPayload) (map[string]string, error) {
	customPayload := payload.CustomPayload()
	data := make(map[string]string, len(customPayload))
	for key, value := range customPayload {
		switch v := value.(type) {
		case string:
			data[key] = v
		case int:
			data[key] = strconv.Itoa(v)
		case int64:
			data[key] = strconv.FormatInt(v, 10)
		case fmt.Stringer:
			data[key] = v.String()
		default:
			encoded, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("encode fcm data field %q: %w", key, err)
			}
			data[key] = string(encoded)
		}
	}
	return data, nil
}

func newFCMResponseError(err error) error {
	if err == nil {
		return nil
	}
	return &FCMResponseError{
		Err:                err,
		Reason:             fcmErrorReason(err),
		invalidDeviceToken: isPermanentFCMInvalidTokenError(err),
	}
}

func fcmErrorReason(err error) string {
	switch {
	case messaging.IsRegistrationTokenNotRegistered(err):
		return "registration-token-not-registered"
	case messaging.IsSenderIDMismatch(err):
		return "sender-id-mismatch"
	case messaging.IsInvalidArgument(err):
		return "invalid-argument"
	case messaging.IsQuotaExceeded(err):
		return "quota-exceeded"
	case messaging.IsUnavailable(err):
		return "unavailable"
	case messaging.IsInternal(err):
		return "internal"
	case messaging.IsThirdPartyAuthError(err):
		return "third-party-auth-error"
	default:
		return ""
	}
}

func isPermanentFCMInvalidTokenError(err error) bool {
	if messaging.IsRegistrationTokenNotRegistered(err) || messaging.IsSenderIDMismatch(err) {
		return true
	}

	if messaging.IsInvalidArgument(err) {
		message := strings.ToLower(err.Error())
		return strings.Contains(message, "registration token") || strings.Contains(message, "device token")
	}
	return false
}
