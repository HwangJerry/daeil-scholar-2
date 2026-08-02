package config

import "testing"

func TestPushConfigDisabledByDefault(t *testing.T) {
	t.Setenv("PUSH_ENABLED", "")
	if Load().Push.Enabled {
		t.Fatal("push must be disabled by default")
	}
}

func TestPushConfigEnabledRequiresBothProviders(t *testing.T) {
	cfg := PushConfig{Enabled: true, FCMProjectID: "project", FCMCredentialsFile: "/credentials/fcm.json"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected incomplete APNs config to fail")
	}
	cfg.APNSTeamID = "team"
	cfg.APNSKeyID = "key"
	cfg.APNSPrivateKeyFile = "/credentials/apns.p8"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("complete config rejected: %v", err)
	}
}

func TestLoadPushConfigUsesCanonicalEnvironmentNames(t *testing.T) {
	t.Setenv("PUSH_ENABLED", "true")
	t.Setenv("FCM_PROJECT_ID", "project")
	t.Setenv("FCM_CREDENTIALS_FILE", "/credentials/fcm.json")
	t.Setenv("APNS_TEAM_ID", "team")
	t.Setenv("APNS_KEY_ID", "key")
	t.Setenv("APNS_PRIVATE_KEY_FILE", "/credentials/apns.p8")
	got := Load().Push
	if !got.Enabled || got.FCMProjectID != "project" || got.FCMCredentialsFile != "/credentials/fcm.json" ||
		got.APNSTeamID != "team" || got.APNSKeyID != "key" || got.APNSPrivateKeyFile != "/credentials/apns.p8" {
		t.Fatalf("push config = %#v", got)
	}
}
