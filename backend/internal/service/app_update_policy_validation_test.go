// app_update_policy_validation_test.go — Guard rails for administrator-submitted policies.
package service

import (
	"errors"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func validIOSPolicy() model.AppUpdatePolicy {
	return model.AppUpdatePolicy{
		ForceEnabled:     true,
		MinBuild:         100,
		RecommendEnabled: true,
		RecommendedBuild: 120,
		MinOSVersion:     "17.0",
		StoreURL:         "https://apps.apple.com/app/id1234567890",
	}
}

func rejectedFields(t *testing.T, err error) map[string]string {
	t.Helper()
	var validation *AppUpdatePolicyValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want validation error", err)
	}
	fields := make(map[string]string, len(validation.Fields))
	for _, field := range validation.Fields {
		fields[field.Field] = field.Reason
	}
	return fields
}

func TestValidateAppUpdatePolicyAcceptsObservedThresholds(t *testing.T) {
	if err := ValidateAppUpdatePolicy(model.AppPlatformIOS, validIOSPolicy(), 120, false); err != nil {
		t.Fatalf("ValidateAppUpdatePolicy() error = %v", err)
	}
}

func TestValidateAppUpdatePolicyRejectsEnabledToggleWithoutThreshold(t *testing.T) {
	policy := validIOSPolicy()
	policy.MinBuild = 0
	policy.RecommendedBuild = 0

	fields := rejectedFields(t, ValidateAppUpdatePolicy(model.AppPlatformIOS, policy, 120, false))

	if _, found := fields["minBuild"]; !found {
		t.Fatalf("fields = %#v, want minBuild rejected", fields)
	}
	if _, found := fields["recommendedBuild"]; !found {
		t.Fatalf("fields = %#v, want recommendedBuild rejected", fields)
	}
}

func TestValidateAppUpdatePolicyRejectsRecommendedBelowMinimum(t *testing.T) {
	policy := validIOSPolicy()
	policy.RecommendedBuild = 90

	fields := rejectedFields(t, ValidateAppUpdatePolicy(model.AppPlatformIOS, policy, 120, false))

	if _, found := fields["recommendedBuild"]; !found {
		t.Fatalf("fields = %#v, want recommendedBuild rejected", fields)
	}
}

// A threshold above every build the API has seen would lock out all users, so it
// takes an explicit confirmation.
func TestValidateAppUpdatePolicyRejectsUnobservedBuildUnlessConfirmed(t *testing.T) {
	policy := validIOSPolicy()

	fields := rejectedFields(t, ValidateAppUpdatePolicy(model.AppPlatformIOS, policy, 50, false))
	if _, found := fields["minBuild"]; !found {
		t.Fatalf("fields = %#v, want minBuild rejected", fields)
	}

	if err := ValidateAppUpdatePolicy(model.AppPlatformIOS, policy, 50, true); err != nil {
		t.Fatalf("confirmed submission error = %v", err)
	}
}

func TestValidateAppUpdatePolicyRequiresIOSStoreURLWhenPromptingUpdates(t *testing.T) {
	policy := validIOSPolicy()
	policy.StoreURL = ""

	fields := rejectedFields(t, ValidateAppUpdatePolicy(model.AppPlatformIOS, policy, 120, false))

	if _, found := fields["storeUrl"]; !found {
		t.Fatalf("fields = %#v, want storeUrl rejected", fields)
	}
}

// The seeded policy ships with an empty store URL; it stays valid until an
// administrator turns a prompt on.
func TestValidateAppUpdatePolicyAllowsEmptyIOSStoreURLWhileDisabled(t *testing.T) {
	policy := model.AppUpdatePolicy{MinOSVersion: "17.0"}

	if err := ValidateAppUpdatePolicy(model.AppPlatformIOS, policy, 0, false); err != nil {
		t.Fatalf("ValidateAppUpdatePolicy() error = %v", err)
	}
}

func TestValidateAppUpdatePolicyRejectsNonAppleStoreURL(t *testing.T) {
	policy := validIOSPolicy()
	policy.StoreURL = "https://example.com/app"

	fields := rejectedFields(t, ValidateAppUpdatePolicy(model.AppPlatformIOS, policy, 120, false))

	if _, found := fields["storeUrl"]; !found {
		t.Fatalf("fields = %#v, want storeUrl rejected", fields)
	}
}

func TestValidateAppUpdatePolicyRejectsStoreURLOnAndroid(t *testing.T) {
	policy := model.AppUpdatePolicy{MinOSVersion: "26", StoreURL: "https://play.google.com/store/apps"}

	fields := rejectedFields(t, ValidateAppUpdatePolicy(model.AppPlatformAndroid, policy, 0, false))

	if _, found := fields["storeUrl"]; !found {
		t.Fatalf("fields = %#v, want storeUrl rejected", fields)
	}
}

// Android clients report Build.VERSION.SDK_INT, so an OS version such as "8.0"
// would never match a real device and would silently disable the notice.
func TestValidateAppUpdatePolicyRequiresAndroidAPILevel(t *testing.T) {
	policy := model.AppUpdatePolicy{MinOSVersion: "8.0"}

	fields := rejectedFields(t, ValidateAppUpdatePolicy(model.AppPlatformAndroid, policy, 0, false))

	if _, found := fields["minOsVersion"]; !found {
		t.Fatalf("fields = %#v, want minOsVersion rejected", fields)
	}
	if err := ValidateAppUpdatePolicy(model.AppPlatformAndroid, model.AppUpdatePolicy{MinOSVersion: "26"}, 0, false); err != nil {
		t.Fatalf("API level policy error = %v", err)
	}
}

func TestValidateAppUpdatePolicyRejectsMalformedOSVersionAndPlatform(t *testing.T) {
	policy := validIOSPolicy()
	policy.MinOSVersion = "17.0.1.2"

	fields := rejectedFields(t, ValidateAppUpdatePolicy("windows", policy, 120, false))

	if _, found := fields["minOsVersion"]; !found {
		t.Fatalf("fields = %#v, want minOsVersion rejected", fields)
	}
	if _, found := fields["platform"]; !found {
		t.Fatalf("fields = %#v, want platform rejected", fields)
	}
}
