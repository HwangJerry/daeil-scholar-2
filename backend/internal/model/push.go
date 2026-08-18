package model

type PushDeviceRegistrationRequest struct {
	Platform    string `json:"platform"`
	DeviceToken string `json:"deviceToken"`
	Locale      string `json:"locale"`
}

type PushDeviceRegistrationResponse struct {
	Status string `json:"status"`
}
