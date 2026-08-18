package service

import (
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/config"
)

func TestMakeAPNsHeadersSetsRequiredHeaders(t *testing.T) {
	sentAt := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	notification := PushNotification{
		BundleID: "kr.dflh.saf.debug",
		Payload:  BuildMessageNewPushPayload(777, 12345, 67890, sentAt),
	}

	headers := makeAPNsHeaders(config.PushConfig{APNsBundleID: "kr.dflh.saf"}, notification)

	if headers.Get("apns-topic") != "kr.dflh.saf.debug" {
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
