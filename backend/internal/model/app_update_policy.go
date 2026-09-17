// app_update_policy.go — Domain model for per-platform app update policies.
package model

import "time"

const (
	AppPlatformIOS     = "ios"
	AppPlatformAndroid = "android"

	// AppUpdatePolicyKeyPrefix namespaces the app_settings rows that hold update
	// policies. Rows with this prefix are writable only through the dedicated
	// policy endpoints, never through the generic app-settings endpoint.
	AppUpdatePolicyKeyPrefix = "app_update_policy_"
)

// AppUpdatePolicy is the policy for one platform, stored as JSON in app_settings
// and published to clients through GET /api/settings/public.
type AppUpdatePolicy struct {
	ForceEnabled     bool   `json:"forceEnabled"`
	MinBuild         int64  `json:"minBuild"`
	RecommendEnabled bool   `json:"recommendEnabled"`
	RecommendedBuild int64  `json:"recommendedBuild"`
	MinOSVersion     string `json:"minOsVersion"`
	StoreURL         string `json:"storeUrl,omitempty"`
}

// AppUpdateDecision is the outcome of evaluating a client against a policy.
type AppUpdateDecision string

const (
	AppUpdateDecisionNone          AppUpdateDecision = "none"
	AppUpdateDecisionRecommend     AppUpdateDecision = "recommend"
	AppUpdateDecisionForce         AppUpdateDecision = "force"
	AppUpdateDecisionUnsupportedOS AppUpdateDecision = "unsupported_os"
)

// AppUpdatePolicyRecord is one stored policy with its audit fields, used by the
// admin API for display and optimistic concurrency.
type AppUpdatePolicyRecord struct {
	Platform  string          `json:"platform"`
	Policy    AppUpdatePolicy `json:"policy"`
	UpdatedAt time.Time       `json:"updatedAt"`
	UpdatedBy *int            `json:"updatedBy"`
	// Unavailable marks a platform whose stored row is missing or undecodable.
	// The other platform is still returned, so an active lockout can always be
	// switched off from the admin screen.
	Unavailable bool `json:"unavailable,omitempty"`
}

// IsSupportedAppPlatform reports whether the platform is one this system manages.
func IsSupportedAppPlatform(platform string) bool {
	return platform == AppPlatformIOS || platform == AppPlatformAndroid
}

// AppUpdatePolicyKey returns the app_settings key holding the platform's policy.
func AppUpdatePolicyKey(platform string) string {
	return AppUpdatePolicyKeyPrefix + platform
}
