package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/rs/zerolog"
	"github.com/sideshow/apns2"
)

type timeoutPushError struct{}

func (timeoutPushError) Error() string   { return "timeout" }
func (timeoutPushError) Timeout() bool   { return true }
func (timeoutPushError) Temporary() bool { return true }

func TestMakeAPNsHeadersSetsRequiredHeaders(t *testing.T) {
	sentAt := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	notification := PushNotification{
		BundleID: "com.daeil.dflhsafv2",
		Payload:  BuildMessageNewPushPayload(777, 12345, 67890, sentAt),
	}

	headers := makeAPNsHeaders(config.PushConfig{APNsBundleID: "com.daeil.dflhsafv2"}, notification)

	if headers.Get("apns-topic") != "com.daeil.dflhsafv2" {
		t.Fatalf("unexpected apns-topic: %q", headers.Get("apns-topic"))
	}
	if headers.Get("apns-push-type") != "alert" {
		t.Fatalf("unexpected apns-push-type: %q", headers.Get("apns-push-type"))
	}
	if headers.Get("apns-priority") != "10" {
		t.Fatalf("unexpected apns-priority: %q", headers.Get("apns-priority"))
	}
	if headers.Get("apns-collapse-id") != "message.new:67890" {
		t.Fatalf("unexpected apns-collapse-id: %q", headers.Get("apns-collapse-id"))
	}
	wantExpiration := "1783036800"
	if headers.Get("apns-expiration") != wantExpiration {
		t.Fatalf("unexpected apns-expiration: got %q want %q", headers.Get("apns-expiration"), wantExpiration)
	}
}

func TestAPNsHostForEnvironmentUsesTokenEnvironment(t *testing.T) {
	if got := apnsHostForEnvironment("sandbox", "production"); got != apnsSandboxHost {
		t.Fatalf("expected sandbox host, got %q", got)
	}
	if got := apnsHostForEnvironment("", "production"); got != apnsProductionHost {
		t.Fatalf("expected production host, got %q", got)
	}
}

func TestNewAPNsPushProviderLoadsSeparateEnvironmentCredentials(t *testing.T) {
	privateKey := makeTestAPNsPrivateKey(t)
	provider, err := NewAPNsPushProvider(config.PushConfig{
		APNSTeamID:      "TEAM123456",
		APNsBundleID:    "com.daeil.dflhsafv2",
		APNsEnvironment: "production",
		APNsSandbox: config.APNsCredentialConfig{
			KeyID:    "SANDBOX01",
			KeyValue: privateKey,
		},
		APNsProduction: config.APNsCredentialConfig{
			KeyID:    "PRODUCTION",
			KeyValue: privateKey,
		},
		APNsRequestTimeout: time.Second,
	}, zerolog.Nop())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	if provider.authTokens["sandbox"].KeyID != "SANDBOX01" {
		t.Fatalf("unexpected sandbox key id: %q", provider.authTokens["sandbox"].KeyID)
	}
	if provider.authTokens["production"].KeyID != "PRODUCTION" {
		t.Fatalf("unexpected production key id: %q", provider.authTokens["production"].KeyID)
	}
	if provider.authTokens["sandbox"] == provider.authTokens["production"] {
		t.Fatal("sandbox and production must not share an auth token cache")
	}
}

func TestNewAPNsPushProviderRejectsPartialEnvironmentCredentials(t *testing.T) {
	_, err := NewAPNsPushProvider(config.PushConfig{
		APNSTeamID:   "TEAM123456",
		APNsBundleID: "com.daeil.dflhsafv2",
		APNsSandbox:  config.APNsCredentialConfig{KeyID: "SANDBOX01"},
	}, zerolog.Nop())
	if err == nil || !strings.Contains(err.Error(), "sandbox private key is required") {
		t.Fatalf("expected incomplete sandbox config error, got %v", err)
	}
}

func TestNewAPNsPushProviderReadsCredentialFileAtStartup(t *testing.T) {
	keyPath := t.TempDir() + "/sandbox.p8"
	if err := os.WriteFile(keyPath, []byte(makeTestAPNsPrivateKey(t)), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	provider, err := NewAPNsPushProvider(config.PushConfig{
		APNSTeamID:   "TEAM123456",
		APNsBundleID: "com.daeil.dflhsafv2",
		APNsSandbox: config.APNsCredentialConfig{
			KeyID:   "SANDBOX01",
			KeyPath: keyPath,
		},
	}, zerolog.Nop())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	if provider.authTokens["sandbox"].KeyID != "SANDBOX01" {
		t.Fatalf("unexpected sandbox key id: %q", provider.authTokens["sandbox"].KeyID)
	}
}

func TestNewAPNsPushProviderRejectsInvalidCredentialFileAtStartup(t *testing.T) {
	keyPath := t.TempDir() + "/sandbox.p8"
	if err := os.WriteFile(keyPath, []byte("not-a-private-key"), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	_, err := NewAPNsPushProvider(config.PushConfig{
		APNSTeamID:   "TEAM123456",
		APNsBundleID: "com.daeil.dflhsafv2",
		APNsSandbox: config.APNsCredentialConfig{
			KeyID:   "SANDBOX01",
			KeyPath: keyPath,
		},
	}, zerolog.Nop())
	if err == nil || !strings.Contains(err.Error(), "parse apns sandbox private key") {
		t.Fatalf("expected invalid sandbox private key error, got %v", err)
	}
}

func TestAPNsPushProviderDoesNotFallbackAcrossEnvironments(t *testing.T) {
	provider, err := NewAPNsPushProvider(config.PushConfig{
		APNSTeamID:      "TEAM123456",
		APNsBundleID:    "com.daeil.dflhsafv2",
		APNsEnvironment: "production",
		APNsSandbox: config.APNsCredentialConfig{
			KeyID:    "SANDBOX01",
			KeyValue: makeTestAPNsPrivateKey(t),
		},
		APNsRequestTimeout: time.Second,
	}, zerolog.Nop())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	err = provider.SendPush(context.Background(), PushNotification{
		DeviceToken:     "production-device-token",
		APNsEnvironment: "production",
	})
	var configErr *APNsConfigurationError
	if !errors.As(err, &configErr) {
		t.Fatalf("expected APNsConfigurationError, got %T: %v", err, err)
	}
	if configErr.Environment != "production" {
		t.Fatalf("unexpected missing environment: %q", configErr.Environment)
	}
}

func TestBuildAPNsNotificationSetsSendPathFields(t *testing.T) {
	sentAt := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	notification := PushNotification{
		DeviceToken: "device-token-1",
		BundleID:    "com.daeil.dflhsafv2",
		Title:       "새 쪽지가 도착했습니다",
		Body:        "새로운 쪽지가 도착했습니다.",
		Payload:     BuildMessageNewPushPayload(777, 12345, 67890, sentAt),
	}
	payloadBytes, err := buildAPNsPayload(notification)
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	apnsNotification := buildAPNsNotification(config.PushConfig{APNsBundleID: "com.daeil.dflhsafv2"}, notification, payloadBytes)

	if apnsNotification.DeviceToken != "device-token-1" {
		t.Fatalf("unexpected device token: %q", apnsNotification.DeviceToken)
	}
	if apnsNotification.Topic != "com.daeil.dflhsafv2" {
		t.Fatalf("unexpected topic: %q", apnsNotification.Topic)
	}
	if apnsNotification.PushType != apns2.PushTypeAlert {
		t.Fatalf("unexpected push type: %q", apnsNotification.PushType)
	}
	if apnsNotification.Priority != apns2.PriorityHigh {
		t.Fatalf("unexpected priority: %d", apnsNotification.Priority)
	}
	if !apnsNotification.Expiration.Equal(sentAt.Add(24 * time.Hour)) {
		t.Fatalf("unexpected expiration: %s", apnsNotification.Expiration)
	}
	if apnsNotification.CollapseID != "message.new:67890" {
		t.Fatalf("unexpected collapse id: %q", apnsNotification.CollapseID)
	}

	gotPayload, ok := apnsNotification.Payload.([]byte)
	if !ok {
		t.Fatalf("expected []byte payload, got %T", apnsNotification.Payload)
	}
	var payload map[string]any
	if err := json.Unmarshal(gotPayload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	assertAPNsPayloadField(t, payload, "event_type", PushEventMessageNew)
	assertAPNsPayloadField(t, payload, "event_id", "message.new:777:67890")
	assertAPNsPayloadField(t, payload, "template_key", "push.message.new")
	assertAPNsPayloadField(t, payload, "ttl_sec", float64(86400))
	assertAPNsPayloadField(t, payload, "collapse_key", "message.new:67890")
	assertAPNsPayloadField(t, payload, "user_id", "67890")
	assertAPNsPayloadField(t, payload, "deep_link", "/messages/12345")
	assertAPNsPayloadField(t, payload, "sent_at", "2026-07-02T00:00:00Z")

	args, ok := payload["args"].(map[string]any)
	if !ok {
		t.Fatalf("expected args object, got %#v", payload["args"])
	}
	if args["sender_seq"] != float64(12345) || args["recvr_seq"] != float64(67890) {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildAPNsNotificationFallsBackToConfigTopic(t *testing.T) {
	notification := PushNotification{
		DeviceToken: "device-token-1",
		Payload:     BuildAdminNoticePushPayload(555, 67890, time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)),
	}

	apnsNotification := buildAPNsNotification(config.PushConfig{APNsBundleID: "com.daeil.dflhsafv2"}, notification, []byte(`{}`))

	if apnsNotification.Topic != "com.daeil.dflhsafv2" {
		t.Fatalf("unexpected fallback topic: %q", apnsNotification.Topic)
	}
}

func TestAPNsErrorClassification(t *testing.T) {
	if !IsInvalidDeviceToken(&APNsResponseError{StatusCode: 400, Reason: "BadDeviceToken"}) {
		t.Fatalf("expected BadDeviceToken invalid")
	}
	if !IsInvalidDeviceToken(&APNsResponseError{StatusCode: 410, Reason: "Unregistered"}) {
		t.Fatalf("expected Unregistered invalid")
	}
	if !IsTransientPushError(&APNsResponseError{StatusCode: 429, Reason: "TooManyRequests"}) {
		t.Fatalf("expected TooManyRequests transient")
	}
	if !IsTransientPushError(&APNsResponseError{StatusCode: 503, Reason: "ServiceUnavailable"}) {
		t.Fatalf("expected ServiceUnavailable transient")
	}
	if !IsTransientPushError(context.DeadlineExceeded) {
		t.Fatalf("expected context deadline transient")
	}
	if !IsTransientPushError(timeoutPushError{}) {
		t.Fatalf("expected net timeout transient")
	}
	if !IsPermanentPushError(&APNsResponseError{StatusCode: 400, Reason: "BadTopic"}) {
		t.Fatalf("expected BadTopic permanent")
	}
	if PushErrorReason(&APNsResponseError{StatusCode: 400, Reason: "BadTopic"}) != "BadTopic" {
		t.Fatalf("unexpected APNs reason")
	}
}

func assertAPNsPayloadField(t *testing.T, payload map[string]any, field string, want any) {
	t.Helper()
	if payload[field] != want {
		t.Fatalf("unexpected payload field %s: got %#v want %#v", field, payload[field], want)
	}
}

func makeTestAPNsPrivateKey(t *testing.T) string {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate EC key: %v", err)
	}
	encoded, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal PKCS8 key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}))
}
