package handler

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAppleNotificationUsesSignInWithApplePayloadField(t *testing.T) {
	var request appleNotificationRequest
	if err := json.NewDecoder(strings.NewReader(`{"payload":"signed-jws"}`)).Decode(&request); err != nil {
		t.Fatal(err)
	}
	if request.notificationPayload() != "signed-jws" {
		t.Fatalf("payload = %q", request.notificationPayload())
	}
}

func TestAppleNotificationKeepsLegacySignedPayloadCompatibility(t *testing.T) {
	var request appleNotificationRequest
	if err := json.NewDecoder(strings.NewReader(`{"signedPayload":"legacy-jws"}`)).Decode(&request); err != nil {
		t.Fatal(err)
	}
	if request.notificationPayload() != "legacy-jws" {
		t.Fatalf("signedPayload = %q", request.notificationPayload())
	}
}
