package model

// PushDeviceRegistration is the canonical device-token registration request.
type PushDeviceRegistration struct {
	Platform        string  `json:"platform"`
	DeviceToken     string  `json:"deviceToken"`
	Locale          string  `json:"locale"`
	APNSEnvironment *string `json:"apnsEnvironment,omitempty"`
	BundleID        *string `json:"bundleId,omitempty"`
}

// PushDeviceUnregistration is the canonical device-token unregistration request.
type PushDeviceUnregistration struct {
	DeviceToken string `json:"deviceToken"`
}

// PushStatusResponse is the closed registration mutation response.
type PushStatusResponse struct {
	Status string `json:"status"`
}

// PushPreferences is the closed account-level push preference response.
type PushPreferences struct {
	MessageEnabled        bool `json:"messageEnabled"`
	MessagePreviewEnabled bool `json:"messagePreviewEnabled"`
}

type PushDeliveryTarget struct {
	Platform        string
	DeviceToken     string
	APNSEnvironment string
	BundleID        string
}

type PushMessagePayload struct {
	Type                string
	EventID             string
	MessageID           string
	ConversationUserSeq string
	SenderUserSeq       string
	SenderName          string
	Preview             string
	CreatedAt           string
}
