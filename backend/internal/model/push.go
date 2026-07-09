package model

import "strings"

type PushDeviceRegistrationRequest struct {
	Platform        string `json:"platform"`
	DeviceToken     string `json:"deviceToken"`
	APNsEnvironment string `json:"apnsEnvironment"`
	BundleID        string `json:"bundleId"`
	Locale          string `json:"locale"`
}

type PushDeviceRegistrationResponse struct {
	Status string `json:"status"`
}

type PushPreferences struct {
	NoticeEnabled  bool `json:"noticeEnabled"`
	MessageEnabled bool `json:"messageEnabled"`
}

type PushPreferencesUpdateRequest struct {
	NoticeEnabled  *bool `json:"noticeEnabled"`
	MessageEnabled *bool `json:"messageEnabled"`
}

func DefaultPushPreferences() PushPreferences {
	return PushPreferences{
		NoticeEnabled:  true,
		MessageEnabled: true,
	}
}

func (r PushPreferencesUpdateRequest) Preferences() (PushPreferences, bool) {
	if r.NoticeEnabled == nil || r.MessageEnabled == nil {
		return PushPreferences{}, false
	}
	return PushPreferences{
		NoticeEnabled:  *r.NoticeEnabled,
		MessageEnabled: *r.MessageEnabled,
	}, true
}

func NormalizeAPNsEnvironment(env string) string {
	env = strings.ToLower(strings.TrimSpace(env))
	switch env {
	case "sandbox", "development", "debug", "dev":
		return "sandbox"
	case "production", "prod", "release", "testflight", "appstore":
		return "production"
	default:
		return ""
	}
}
