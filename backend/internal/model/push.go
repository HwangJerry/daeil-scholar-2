package model

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
