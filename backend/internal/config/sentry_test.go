package config

import "testing"

func TestLoadReadsSentryMonitoringConfig(t *testing.T) {
	t.Setenv("SENTRY_AUTH_TOKEN", "test-token")
	t.Setenv("SENTRY_ORG", "test-org")
	t.Setenv("SENTRY_IOS_PROJECT", "test-ios")
	t.Setenv("SENTRY_ANDROID_PROJECT", "test-android")

	cfg := Load()
	if !cfg.Sentry.Configured() {
		t.Fatal("sentry config should be configured")
	}
	if cfg.Sentry.Organization != "test-org" || cfg.Sentry.IOSProject != "test-ios" || cfg.Sentry.AndroidProject != "test-android" {
		t.Fatalf("sentry project config = %#v", cfg.Sentry)
	}
}

func TestSentryConfigRequiresEveryValue(t *testing.T) {
	cfg := SentryConfig{AuthToken: "token", Organization: "org", IOSProject: "ios"}
	if cfg.Configured() {
		t.Fatal("incomplete sentry config must not be configured")
	}
}
