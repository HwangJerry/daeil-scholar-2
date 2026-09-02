// mobile_app_event.go — Mobile application business-event analytics models.
package model

import "time"

const (
	MobilePlatformIOS     = "ios"
	MobilePlatformAndroid = "android"

	MobileEventSignupStart    = "signup_start"
	MobileEventSignupComplete = "signup_complete"
	MobileEventApplyComplete  = "apply_complete"
)

// MobileAppEvent is one client-side business event. UserID is populated from
// the authenticated request context and is intentionally not client-writable.
type MobileAppEvent struct {
	Platform    string    `json:"platform" db:"PLATFORM"`
	EventType   string    `json:"eventType" db:"EVENT_TYPE"`
	UserID      *int      `json:"-" db:"USER_ID"`
	AppVersion  string    `json:"appVersion" db:"APP_VERSION"`
	OSVersion   string    `json:"osVersion" db:"OS_VERSION"`
	DeviceModel *string   `json:"deviceModel,omitempty" db:"DEVICE_MODEL"`
	OccurredAt  time.Time `json:"occurredAt" db:"OCCURRED_AT"`
	CreatedAt   time.Time `json:"-" db:"CREATED_AT"`
}

type MobileAppEventBatchRequest struct {
	Events []MobileAppEvent `json:"events"`
}

type MobileAppEventBatchResponse struct {
	Accepted int `json:"accepted"`
}

type MobileAppEventSummary struct {
	Platform  string `json:"platform" db:"PLATFORM"`
	EventType string `json:"eventType" db:"EVENT_TYPE"`
	Count     uint64 `json:"count" db:"EVENT_COUNT"`
}

type MobileAppEventSummaryFilter struct {
	From        time.Time
	ToExclusive time.Time
	Platform    string
	EventType   string
}

type MobileAppEventSummaryResponse struct {
	From      string                  `json:"from"`
	To        string                  `json:"to"`
	Platform  string                  `json:"platform,omitempty"`
	EventType string                  `json:"eventType,omitempty"`
	Items     []MobileAppEventSummary `json:"items"`
}
