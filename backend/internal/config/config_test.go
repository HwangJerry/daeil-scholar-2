package config

import "testing"

var apnsEnvironmentKeys = []string{
	"APNS_KEY_ID",
	"APNS_TEAM_ID",
	"APNS_BUNDLE_ID",
	"APNS_PRIVATE_KEY_PATH",
	"APNS_PRIVATE_KEY",
	"APNS_ENVIRONMENT",
	"APNS_SANDBOX_KEY_ID",
	"APNS_SANDBOX_PRIVATE_KEY_PATH",
	"APNS_SANDBOX_PRIVATE_KEY",
	"APNS_PRODUCTION_KEY_ID",
	"APNS_PRODUCTION_PRIVATE_KEY_PATH",
	"APNS_PRODUCTION_PRIVATE_KEY",
	"PUSH_APNS_KEY_ID",
	"PUSH_APNS_TEAM_ID",
	"PUSH_APNS_BUNDLE_ID",
	"PUSH_APNS_KEY_PATH",
	"PUSH_APNS_KEY_VALUE",
	"PUSH_APNS_USE_SANDBOX",
}

func clearAPNsEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range apnsEnvironmentKeys {
		t.Setenv(key, "")
	}
}

func TestLoadReadsEnvironmentSpecificAPNsCredentials(t *testing.T) {
	clearAPNsEnvironment(t)
	t.Setenv("APNS_TEAM_ID", "TEAM123")
	t.Setenv("APNS_BUNDLE_ID", "com.example.app")
	t.Setenv("APNS_SANDBOX_KEY_ID", "SANDBOX1")
	t.Setenv("APNS_SANDBOX_PRIVATE_KEY_PATH", "/secrets/sandbox.p8")
	t.Setenv("APNS_PRODUCTION_KEY_ID", "PROD1234")
	t.Setenv("APNS_PRODUCTION_PRIVATE_KEY_PATH", "/secrets/production.p8")

	push := Load().Push

	if push.APNsSandbox.KeyID != "SANDBOX1" || push.APNsSandbox.KeyPath != "/secrets/sandbox.p8" {
		t.Fatalf("unexpected sandbox credentials: %#v", push.APNsSandbox)
	}
	if push.APNsProduction.KeyID != "PROD1234" || push.APNsProduction.KeyPath != "/secrets/production.p8" {
		t.Fatalf("unexpected production credentials: %#v", push.APNsProduction)
	}
}

func TestLoadMapsLegacyAPNsCredentialsOnlyToDefaultEnvironment(t *testing.T) {
	clearAPNsEnvironment(t)
	t.Setenv("APNS_ENVIRONMENT", "development")
	t.Setenv("APNS_KEY_ID", "LEGACY01")
	t.Setenv("APNS_PRIVATE_KEY_PATH", "/secrets/legacy.p8")

	push := Load().Push

	if push.APNsEnvironment != "sandbox" {
		t.Fatalf("unexpected default environment: %q", push.APNsEnvironment)
	}
	if push.APNsSandbox.KeyID != "LEGACY01" || push.APNsSandbox.KeyPath != "/secrets/legacy.p8" {
		t.Fatalf("legacy credentials were not mapped to sandbox: %#v", push.APNsSandbox)
	}
	if push.APNsProduction.Configured() {
		t.Fatalf("legacy credentials must not be shared with production: %#v", push.APNsProduction)
	}
}

func TestLoadEnvironmentSpecificAPNsCredentialsOverrideLegacySet(t *testing.T) {
	clearAPNsEnvironment(t)
	t.Setenv("APNS_ENVIRONMENT", "sandbox")
	t.Setenv("APNS_KEY_ID", "LEGACY01")
	t.Setenv("APNS_PRIVATE_KEY_PATH", "/secrets/legacy.p8")
	t.Setenv("APNS_SANDBOX_KEY_ID", "EXPLICIT")

	push := Load().Push

	if push.APNsSandbox.KeyID != "EXPLICIT" {
		t.Fatalf("unexpected sandbox key id: %q", push.APNsSandbox.KeyID)
	}
	if push.APNsSandbox.KeyPath != "" {
		t.Fatalf("partial explicit config must not be completed from legacy config: %#v", push.APNsSandbox)
	}
}

func TestLoadSupportsLegacySandboxBoolean(t *testing.T) {
	clearAPNsEnvironment(t)
	t.Setenv("PUSH_APNS_USE_SANDBOX", "true")
	t.Setenv("PUSH_APNS_KEY_ID", "LEGACY01")
	t.Setenv("PUSH_APNS_KEY_VALUE", "private-key")

	push := Load().Push

	if push.APNsEnvironment != "sandbox" {
		t.Fatalf("unexpected default environment: %q", push.APNsEnvironment)
	}
	if push.APNsSandbox.KeyID != "LEGACY01" || push.APNsSandbox.KeyValue != "private-key" {
		t.Fatalf("legacy PUSH_APNS variables were not loaded: %#v", push.APNsSandbox)
	}
}

func TestHasAnyAPNsCredentialsIncludesPartialSets(t *testing.T) {
	push := PushConfig{APNsProduction: APNsCredentialConfig{KeyID: "PROD1234"}}
	if !push.HasAnyAPNsCredentials() {
		t.Fatal("partial credential sets must trigger startup validation")
	}
}
