package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/config"
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
