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
	RecipientUserSeq    string
	VerificationStatus  VerificationStatus
	Type                string
	EventID             string
	MessageID           string
	ConversationUserSeq string
	SenderUserSeq       string
	SenderName          string
	// Title and Body are the admin-editable text the operating system displays.
	// They are kept apart from SenderName and Preview, which stay the real
	// sender's name and the raw message snippet the apps route, label, and list
	// conversations by.
	Title           string
	Body            string
	Preview         string
	CreatedAt       string
	TemplateKey     string
	TemplateVersion int
}
