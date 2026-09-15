// app_update_policy_validation.go — Field validation for admin-submitted update policies.
package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
)

const iosStoreURLPrefix = "https://apps.apple.com/"

var (
	dottedOSVersionPattern = regexp.MustCompile(`^\d+(\.\d+){0,2}$`)
	apiLevelPattern        = regexp.MustCompile(`^\d+$`)
)

// AppUpdatePolicyFieldError names one rejected field so the admin UI can point
// at the input that needs fixing.
type AppUpdatePolicyFieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// AppUpdatePolicyValidationError carries every field rejected by one submission.
type AppUpdatePolicyValidationError struct {
	Fields []AppUpdatePolicyFieldError
}

func (e *AppUpdatePolicyValidationError) Error() string {
	reasons := make([]string, 0, len(e.Fields))
	for _, field := range e.Fields {
		reasons = append(reasons, fmt.Sprintf("%s: %s", field.Field, field.Reason))
	}
	return "invalid app update policy (" + strings.Join(reasons, "; ") + ")"
}

// ValidateAppUpdatePolicy rejects submissions that could lock users out.
//
// latestObservedBuild is the highest build the API has actually seen from this
// platform; thresholds above it would block everyone, so they are refused unless
// the administrator explicitly confirmed an unobserved build.
func ValidateAppUpdatePolicy(
	platform string,
	policy model.AppUpdatePolicy,
	latestObservedBuild int64,
	allowUnobservedBuild bool,
) error {
	fields := make([]AppUpdatePolicyFieldError, 0)
	add := func(field, reason string) {
		fields = append(fields, AppUpdatePolicyFieldError{Field: field, Reason: reason})
	}

	if !model.IsSupportedAppPlatform(platform) {
		add("platform", "지원하지 않는 플랫폼입니다")
	}
	if policy.MinBuild < 0 {
		add("minBuild", "0 이상의 정수여야 합니다")
	}
	if policy.RecommendedBuild < 0 {
		add("recommendedBuild", "0 이상의 정수여야 합니다")
	}
	if policy.ForceEnabled && policy.MinBuild <= 0 {
		add("minBuild", "강제 업데이트를 사용하려면 최소 빌드를 지정해야 합니다")
	}
	if policy.RecommendEnabled && policy.RecommendedBuild <= 0 {
		add("recommendedBuild", "권장 업데이트를 사용하려면 권장 빌드를 지정해야 합니다")
	}
	if policy.ForceEnabled && policy.RecommendEnabled && policy.RecommendedBuild < policy.MinBuild {
		add("recommendedBuild", "권장 빌드는 최소 빌드 이상이어야 합니다")
	}
	if !allowUnobservedBuild {
		if policy.ForceEnabled && policy.MinBuild > latestObservedBuild {
			add("minBuild", "이 플랫폼에서 확인된 적 없는 빌드입니다")
		}
		if policy.RecommendEnabled && policy.RecommendedBuild > latestObservedBuild {
			add("recommendedBuild", "이 플랫폼에서 확인된 적 없는 빌드입니다")
		}
	}
	if platform == model.AppPlatformAndroid {
		// An operator entering "8.0" (the OS version) instead of "26" (the API
		// level the client reports) would silently disable the notice.
		if !apiLevelPattern.MatchString(policy.MinOSVersion) {
			add("minOsVersion", "Android는 API 레벨 정수여야 합니다 (예: 26)")
		}
	} else if !dottedOSVersionPattern.MatchString(policy.MinOSVersion) {
		add("minOsVersion", "17.0 형식이어야 합니다")
	}
	if platform == model.AppPlatformIOS {
		validateIOSStoreURL(policy, add)
	} else if strings.TrimSpace(policy.StoreURL) != "" {
		add("storeUrl", "Android 정책에는 스토어 URL을 사용하지 않습니다")
	}

	if len(fields) == 0 {
		return nil
	}
	return &AppUpdatePolicyValidationError{Fields: fields}
}

// validateIOSStoreURL keeps a forced iOS client from reaching a screen whose
// update button has nowhere to go.
func validateIOSStoreURL(policy model.AppUpdatePolicy, add func(field, reason string)) {
	storeURL := strings.TrimSpace(policy.StoreURL)
	if storeURL == "" {
		if policy.ForceEnabled || policy.RecommendEnabled {
			add("storeUrl", "업데이트 안내를 사용하려면 App Store URL이 필요합니다")
		}
		return
	}
	if !strings.HasPrefix(storeURL, iosStoreURLPrefix) {
		add("storeUrl", "https://apps.apple.com/ 으로 시작해야 합니다")
	}
}
