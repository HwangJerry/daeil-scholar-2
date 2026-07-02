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
