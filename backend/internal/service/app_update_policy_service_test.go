// app_update_policy_service_test.go — Policy read, write, and concurrency behavior.
package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

type appUpdatePolicyStoreStub struct {
	settings      map[string]model.AppSetting
	conflict      bool
	saveCalls     int
	savedKey      string
	savedPlatform string
	savedJSON     string
	savedBy       int
	savedExpected time.Time
	history       []model.AppUpdatePolicyHistoryEntry
}

func (s *appUpdatePolicyStoreStub) GetPolicySetting(key string) (model.AppSetting, error) {
	setting, found := s.settings[key]
	if !found {
		return model.AppSetting{}, sql.ErrNoRows
	}
	return setting, nil
}

func (s *appUpdatePolicyStoreStub) SavePolicy(key, platform, afterJSON string, updatedBy int, expectedUpdatedAt time.Time) (bool, error) {
	s.saveCalls++
	if s.conflict {
		return true, nil
	}
	s.savedKey, s.savedPlatform, s.savedJSON = key, platform, afterJSON
	s.savedBy, s.savedExpected = updatedBy, expectedUpdatedAt
	return false, nil
}

func (s *appUpdatePolicyStoreStub) ListHistory(platform string, _ int) ([]model.AppUpdatePolicyHistoryEntry, error) {
	entries := make([]model.AppUpdatePolicyHistoryEntry, 0)
	for _, entry := range s.history {
		if entry.Platform == platform {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

type publicSettingsCacheStub struct {
	values        map[string]string
	invalidations int
	err           error
}

func (s *publicSettingsCacheStub) GetPublicSettings() (map[string]string, error) {
	return s.values, s.err
}

func (s *publicSettingsCacheStub) InvalidatePublicSettingsCache() {
	s.invalidations++
}

type latestBuildStub struct {
	builds map[string]int64
}

func (s *latestBuildStub) LatestBuild(platform string) (int64, error) {
	return s.builds[platform], nil
}

func newPolicyServiceFixture() (*AppUpdatePolicyService, *appUpdatePolicyStoreStub, *publicSettingsCacheStub) {
	updatedAt := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	store := &appUpdatePolicyStoreStub{settings: map[string]model.AppSetting{
		"app_update_policy_ios": {
			Key:       "app_update_policy_ios",
			Value:     `{"forceEnabled":false,"minBuild":0,"recommendEnabled":false,"recommendedBuild":0,"minOsVersion":"17.0","storeUrl":""}`,
			UpdatedAt: updatedAt,
		},
		"app_update_policy_android": {
			Key:       "app_update_policy_android",
			Value:     `{"forceEnabled":false,"minBuild":0,"recommendEnabled":false,"recommendedBuild":0,"minOsVersion":"26"}`,
			UpdatedAt: updatedAt,
		},
	}}
	settings := &publicSettingsCacheStub{values: map[string]string{
		"app_update_policy_ios":     store.settings["app_update_policy_ios"].Value,
		"app_update_policy_android": store.settings["app_update_policy_android"].Value,
	}}
	builds := &latestBuildStub{builds: map[string]int64{
		model.AppPlatformIOS:     202609151230,
		model.AppPlatformAndroid: 260915001,
	}}
	return NewAppUpdatePolicyService(store, settings, builds), store, settings
}

func enabledIOSPolicy() model.AppUpdatePolicy {
	return model.AppUpdatePolicy{
		ForceEnabled: true,
		MinBuild:     202609151230,
		MinOSVersion: "17.0",
		StoreURL:     "https://apps.apple.com/app/id1234567890",
	}
}

func TestAppUpdatePolicyServiceReadsPolicyFromPublicSettings(t *testing.T) {
	service, _, _ := newPolicyServiceFixture()

	policy, err := service.GetPolicy(model.AppPlatformAndroid)
	if err != nil {
		t.Fatalf("GetPolicy() error = %v", err)
	}
	if policy.ForceEnabled || policy.MinOSVersion != "26" {
		t.Fatalf("policy = %#v", policy)
	}
}

func TestAppUpdatePolicyServiceRejectsUnknownPlatform(t *testing.T) {
	service, _, _ := newPolicyServiceFixture()

	if _, err := service.GetPolicy("windows"); !errors.Is(err, ErrUnsupportedAppUpdatePlatform) {
		t.Fatalf("GetPolicy() error = %v", err)
	}
	if err := service.SavePolicy("windows", enabledIOSPolicy(), time.Now(), false, 1); !errors.Is(err, ErrUnsupportedAppUpdatePlatform) {
		t.Fatalf("SavePolicy() error = %v", err)
	}
}

func TestAppUpdatePolicyServiceSaveWritesOnePlatformAndInvalidatesCache(t *testing.T) {
	service, store, settings := newPolicyServiceFixture()
	expected := store.settings["app_update_policy_ios"].UpdatedAt

	if err := service.SavePolicy(model.AppPlatformIOS, enabledIOSPolicy(), expected, false, 7); err != nil {
		t.Fatalf("SavePolicy() error = %v", err)
	}

	if store.savedKey != "app_update_policy_ios" || store.savedPlatform != model.AppPlatformIOS {
		t.Fatalf("saved key = %q, platform = %q", store.savedKey, store.savedPlatform)
	}
	if store.savedBy != 7 || !store.savedExpected.Equal(expected) {
		t.Fatalf("saved by = %d, expected updatedAt = %v", store.savedBy, store.savedExpected)
	}
	var saved model.AppUpdatePolicy
	if err := json.Unmarshal([]byte(store.savedJSON), &saved); err != nil {
		t.Fatalf("saved json = %q: %v", store.savedJSON, err)
	}
	if !saved.ForceEnabled || saved.MinBuild != 202609151230 {
		t.Fatalf("saved policy = %#v", saved)
	}
	if settings.invalidations != 1 {
		t.Fatalf("cache invalidations = %d, want 1", settings.invalidations)
	}
}

// Android policies never carry a store URL, so the encoded payload must omit it.
func TestAppUpdatePolicyServiceSaveOmitsEmptyStoreURL(t *testing.T) {
	service, store, _ := newPolicyServiceFixture()
	policy := model.AppUpdatePolicy{
		ForceEnabled: true,
		MinBuild:     260915001,
		MinOSVersion: "26",
	}

	if err := service.SavePolicy(model.AppPlatformAndroid, policy, store.settings["app_update_policy_android"].UpdatedAt, false, 7); err != nil {
		t.Fatalf("SavePolicy() error = %v", err)
	}
	if strings.Contains(store.savedJSON, "storeUrl") {
		t.Fatalf("saved json = %q, want no storeUrl", store.savedJSON)
	}
}

func TestAppUpdatePolicyServiceSaveReportsConflictWithoutInvalidatingCache(t *testing.T) {
	service, store, settings := newPolicyServiceFixture()
	store.conflict = true

	err := service.SavePolicy(model.AppPlatformIOS, enabledIOSPolicy(), time.Now(), false, 7)

	if !errors.Is(err, ErrAppUpdatePolicyConflict) {
		t.Fatalf("SavePolicy() error = %v", err)
	}
	if settings.invalidations != 0 {
		t.Fatalf("cache invalidations = %d, want 0", settings.invalidations)
	}
}

func TestAppUpdatePolicyServiceSaveRejectsInvalidPolicyBeforeWriting(t *testing.T) {
	service, store, settings := newPolicyServiceFixture()
	policy := enabledIOSPolicy()
	policy.MinBuild = 999999999999

	err := service.SavePolicy(model.AppPlatformIOS, policy, time.Now(), false, 7)

	var validation *AppUpdatePolicyValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("SavePolicy() error = %v, want validation error", err)
	}
	if store.saveCalls != 0 || settings.invalidations != 0 {
		t.Fatalf("save calls = %d, invalidations = %d", store.saveCalls, settings.invalidations)
	}
}

func TestAppUpdatePolicyServiceListPoliciesReturnsBothPlatforms(t *testing.T) {
	service, _, _ := newPolicyServiceFixture()

	records, err := service.ListPolicies()
	if err != nil {
		t.Fatalf("ListPolicies() error = %v", err)
	}
	if len(records) != 2 || records[0].Platform != model.AppPlatformIOS || records[1].Platform != model.AppPlatformAndroid {
		t.Fatalf("records = %#v", records)
	}
	if records[0].UpdatedAt.IsZero() {
		t.Fatalf("record = %#v, want audit fields", records[0])
	}
}

// A broken row for one platform must not hide the other: the admin screen is
// how an active force lockout gets switched off.
func TestAppUpdatePolicyServiceListPoliciesSurvivesOneBrokenRow(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(store *appUpdatePolicyStoreStub)
	}{
		{"missing row", func(store *appUpdatePolicyStoreStub) {
			delete(store.settings, "app_update_policy_android")
		}},
		{"undecodable value", func(store *appUpdatePolicyStoreStub) {
			setting := store.settings["app_update_policy_android"]
			setting.Value = "{not json"
			store.settings["app_update_policy_android"] = setting
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, store, _ := newPolicyServiceFixture()
			test.mutate(store)

			records, err := service.ListPolicies()
			if err != nil {
				t.Fatalf("ListPolicies() error = %v", err)
			}
			if len(records) != 2 {
				t.Fatalf("records = %#v", records)
			}
			if records[0].Unavailable || records[0].Platform != model.AppPlatformIOS {
				t.Fatalf("ios record = %#v, want readable", records[0])
			}
			if !records[1].Unavailable {
				t.Fatalf("android record = %#v, want unavailable", records[1])
			}
		})
	}
}

func TestAppUpdatePolicyServiceListHistoryFiltersByPlatform(t *testing.T) {
	service, store, _ := newPolicyServiceFixture()
	store.history = []model.AppUpdatePolicyHistoryEntry{
		{Platform: model.AppPlatformIOS, AfterJSON: "{}"},
		{Platform: model.AppPlatformAndroid, AfterJSON: "{}"},
	}

	entries, err := service.ListHistory(model.AppPlatformIOS)
	if err != nil {
		t.Fatalf("ListHistory() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Platform != model.AppPlatformIOS {
		t.Fatalf("entries = %#v", entries)
	}
}
