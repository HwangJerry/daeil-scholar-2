package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/rs/zerolog"
)

type fakeFCMMessagingClient struct {
	message *messaging.Message
	err     error
}

func (f *fakeFCMMessagingClient) Send(_ context.Context, message *messaging.Message) (string, error) {
	f.message = message
	return "projects/demo/messages/1", f.err
}

func TestFCMPushProviderBuildsAndroidMessage(t *testing.T) {
	client := &fakeFCMMessagingClient{}
	provider := NewFCMPushProviderForClient(client, time.Second, zerolog.Nop())
	sentAt := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	notification := PushNotification{
		DeviceToken: " fcm-token ",
		Platform:    "android",
		Title:       "새 쪽지가 도착했습니다",
		Body:        "새로운 쪽지가 도착했습니다.",
		Payload:     BuildMessageNewPushPayload(777, 12345, 67890, sentAt),
	}

	if err := provider.SendPush(context.Background(), notification); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.message == nil {
		t.Fatal("expected fcm message to be sent")
	}
	if client.message.Token != "fcm-token" {
		t.Fatalf("unexpected token: %q", client.message.Token)
	}
	if client.message.Notification.Title != notification.Title || client.message.Notification.Body != notification.Body {
		t.Fatalf("unexpected notification: %#v", client.message.Notification)
	}
	if client.message.Android == nil || client.message.Android.CollapseKey != "message.new:67890" ||
		client.message.Android.Priority != "high" {
		t.Fatalf("unexpected android config: %#v", client.message.Android)
	}
	if client.message.Data["event_type"] != PushEventMessageNew ||
		client.message.Data["event"] != PushEventMessageNew ||
		client.message.Data["template_version"] != "1" ||
		client.message.Data["ttl_sec"] != "86400" ||
		client.message.Data["sent_at"] != "2026-07-02T00:00:00Z" {
		t.Fatalf("unexpected fcm data: %#v", client.message.Data)
	}

	var args map[string]int
	if err := json.Unmarshal([]byte(client.message.Data["args"]), &args); err != nil {
		t.Fatalf("args should be json: %v", err)
	}
	if args["sender_seq"] != 12345 || args["recvr_seq"] != 67890 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestFCMPushProviderNormalizesNonPositiveRequestTimeout(t *testing.T) {
	provider := NewFCMPushProviderForClient(&fakeFCMMessagingClient{}, 0, zerolog.Nop())
	if provider.requestTimeout <= 0 {
		t.Fatal("FCM provider accepted an unbounded request timeout")
	}
}

func TestFCMResponseErrorInvalidTokenContract(t *testing.T) {
	err := &FCMResponseError{
		Err:                errors.New("registration token is not registered"),
		Reason:             "registration-token-not-registered",
		invalidDeviceToken: true,
	}

	if !isInvalidPushTokenError(err) {
		t.Fatal("expected FCM invalid token error to satisfy invalid token contract")
	}
}

func TestFCMResponseErrorTransientIsNotInvalidToken(t *testing.T) {
	err := &FCMResponseError{
		Err:                errors.New("backend unavailable"),
		Reason:             "unavailable",
		invalidDeviceToken: false,
	}

	if isInvalidPushTokenError(err) {
		t.Fatal("expected FCM transient error to stay active")
	}
}
