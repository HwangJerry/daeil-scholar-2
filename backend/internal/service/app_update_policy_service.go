// app_update_policy_service.go — Reads and writes per-platform app update policies.
package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

const appUpdatePolicyHistoryLimit = 50

var (
	ErrAppUpdatePolicyNotFound      = errors.New("app update policy not found")
	ErrAppUpdatePolicyConflict      = errors.New("app update policy changed since it was read")
	ErrUnsupportedAppUpdatePlatform = errors.New("unsupported app update platform")
)

// AppUpdatePolicyStore persists policies in app_settings together with an audit
// trail. Writes are transactional: the setting row and its history entry land
// together or not at all.
type AppUpdatePolicyStore interface {
	GetPolicySetting(key string) (model.AppSetting, error)
	// SavePolicy reports whether the row changed since expectedUpdatedAt was read;
	// a conflict writes nothing.
	SavePolicy(key, platform, afterJSON string, updatedBy int, expectedUpdatedAt time.Time) (bool, error)
	ListHistory(platform string, limit int) ([]model.AppUpdatePolicyHistoryEntry, error)
}

// PublicSettingsCache is the read path shared with clients: policies reach apps
// through the cached public settings payload, so writes must invalidate it.
type PublicSettingsCache interface {
	GetPublicSettings() (map[string]string, error)
	InvalidatePublicSettingsCache()
}

// LatestClientBuildProvider reports the highest build the API has seen from a
// platform, which bounds the thresholds an administrator may set.
type LatestClientBuildProvider interface {
	LatestBuild(platform string) (int64, error)
}

type AppUpdatePolicyService struct {
	store    AppUpdatePolicyStore
	settings PublicSettingsCache
	builds   LatestClientBuildProvider
}

func NewAppUpdatePolicyService(
	store AppUpdatePolicyStore,
	settings PublicSettingsCache,
	builds LatestClientBuildProvider,
) *AppUpdatePolicyService {
	return &AppUpdatePolicyService{store: store, settings: settings, builds: builds}
}

// GetPolicy reads one platform's policy from the cached public settings. This is
// the hot path used by the request gate.
func (s *AppUpdatePolicyService) GetPolicy(platform string) (model.AppUpdatePolicy, error) {
	if !model.IsSupportedAppPlatform(platform) {
		return model.AppUpdatePolicy{}, ErrUnsupportedAppUpdatePlatform
	}
	settings, err := s.settings.GetPublicSettings()
	if err != nil {
		return model.AppUpdatePolicy{}, err
	}
	raw, found := settings[model.AppUpdatePolicyKey(platform)]
	if !found {
		return model.AppUpdatePolicy{}, ErrAppUpdatePolicyNotFound
	}
	return decodeAppUpdatePolicy(raw)
}

// ListPolicies returns both platforms with their audit fields for the admin UI.
func (s *AppUpdatePolicyService) ListPolicies() ([]model.AppUpdatePolicyRecord, error) {
	platforms := []string{model.AppPlatformIOS, model.AppPlatformAndroid}
	records := make([]model.AppUpdatePolicyRecord, 0, len(platforms))
	for _, platform := range platforms {
		record, err := s.getPolicyRecord(platform)
		if err != nil {
			// One broken row must not hide the other platform: the admin screen
			// is how an active force lockout gets switched off.
			if !errors.Is(err, ErrAppUpdatePolicyNotFound) && !isPolicyDecodeError(err) {
				return nil, err
			}
			record = model.AppUpdatePolicyRecord{Platform: platform, Unavailable: true}
		}
		records = append(records, record)
	}
	return records, nil
}

// SavePolicy validates and stores one platform's policy. expectedUpdatedAt is the
// UPDATED_AT the administrator last read; a mismatch means someone else saved
// first and the write is refused instead of overwriting their change.
func (s *AppUpdatePolicyService) SavePolicy(
	platform string,
	policy model.AppUpdatePolicy,
	expectedUpdatedAt time.Time,
	allowUnobservedBuild bool,
	updatedBy int,
) error {
	if !model.IsSupportedAppPlatform(platform) {
		return ErrUnsupportedAppUpdatePlatform
	}

	latestObservedBuild := int64(0)
	if !allowUnobservedBuild {
		observed, err := s.builds.LatestBuild(platform)
		if err != nil {
			return err
		}
		latestObservedBuild = observed
	}
	if err := ValidateAppUpdatePolicy(platform, policy, latestObservedBuild, allowUnobservedBuild); err != nil {
		return err
	}

	encoded, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	conflict, err := s.store.SavePolicy(
		model.AppUpdatePolicyKey(platform),
		platform,
		string(encoded),
		updatedBy,
		expectedUpdatedAt,
	)
	if err != nil {
		return translateAppUpdatePolicyStoreError(err)
	}
	if conflict {
		return ErrAppUpdatePolicyConflict
	}

	// Clients and the request gate both read policies through the public
	// settings cache, so a stale entry would delay the change by its TTL.
	s.settings.InvalidatePublicSettingsCache()
	return nil
}

// ListHistory returns recent changes for one platform, newest first.
func (s *AppUpdatePolicyService) ListHistory(platform string) ([]model.AppUpdatePolicyHistoryEntry, error) {
	if !model.IsSupportedAppPlatform(platform) {
		return nil, ErrUnsupportedAppUpdatePlatform
	}
	return s.store.ListHistory(platform, appUpdatePolicyHistoryLimit)
}

func (s *AppUpdatePolicyService) getPolicyRecord(platform string) (model.AppUpdatePolicyRecord, error) {
	setting, err := s.store.GetPolicySetting(model.AppUpdatePolicyKey(platform))
	if err != nil {
		return model.AppUpdatePolicyRecord{}, translateAppUpdatePolicyStoreError(err)
	}
	policy, err := decodeAppUpdatePolicy(setting.Value)
	if err != nil {
		return model.AppUpdatePolicyRecord{}, err
	}
	return model.AppUpdatePolicyRecord{
		Platform:  platform,
		Policy:    policy,
		UpdatedAt: setting.UpdatedAt,
		UpdatedBy: setting.UpdatedBy,
	}, nil
}

// translateAppUpdatePolicyStoreError turns a missing row into a domain error so
// handlers answer 404 instead of 500.
func translateAppUpdatePolicyStoreError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAppUpdatePolicyNotFound
	}
	return err
}

func decodeAppUpdatePolicy(raw string) (model.AppUpdatePolicy, error) {
	var policy model.AppUpdatePolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return model.AppUpdatePolicy{}, err
	}
	return policy, nil
}

// isPolicyDecodeError reports whether a stored policy value failed to parse,
// which is a data problem for one platform rather than a server failure.
func isPolicyDecodeError(err error) bool {
	var syntax *json.SyntaxError
	var unmarshalType *json.UnmarshalTypeError
	return errors.As(err, &syntax) || errors.As(err, &unmarshalType)
}
