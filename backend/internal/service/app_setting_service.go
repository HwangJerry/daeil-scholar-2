// app_setting_service.go — Application-setting validation, caching, and orchestration.
package service

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
)

const (
	appSettingsPublicCacheKey = "app_settings_public"
	appSettingsPublicCacheTTL = 5 * time.Minute
	maxAppSettingKeyRunes     = 100
	maxAppSettingValueBytes   = 65535
)

var (
	ErrAppSettingNotFound     = errors.New("app setting not found")
	ErrAppSettingReadOnly     = errors.New("app setting is managed by a dedicated endpoint")
	ErrInvalidAppSettingKey   = errors.New("invalid app setting key")
	ErrInvalidAppSettingValue = errors.New("invalid app setting value")
)

type AppSettingStore interface {
	ListAll() ([]model.AppSetting, error)
	ListPublic() ([]model.AppSetting, error)
	UpdateValue(key, value string, updatedBy int) (bool, error)
}

type AppSettingService struct {
	store AppSettingStore
	cache *cache.Cache
}

func NewAppSettingService(store AppSettingStore, cacheStore *cache.Cache) *AppSettingService {
	return &AppSettingService{store: store, cache: cacheStore}
}

func (s *AppSettingService) ListSettings() ([]model.AppSetting, error) {
	return s.store.ListAll()
}

func (s *AppSettingService) GetPublicSettings() (map[string]string, error) {
	if s.cache != nil {
		if cached, found := s.cache.Get(appSettingsPublicCacheKey); found {
			if settings, ok := cached.(map[string]string); ok {
				return clonePublicSettings(settings), nil
			}
		}
	}

	rows, err := s.store.ListPublic()
	if err != nil {
		return nil, err
	}
	settings := make(map[string]string, len(rows))
	for _, row := range rows {
		settings[row.Key] = row.Value
	}
	if s.cache != nil {
		s.cache.Set(appSettingsPublicCacheKey, clonePublicSettings(settings), appSettingsPublicCacheTTL)
	}
	return settings, nil
}

func (s *AppSettingService) UpdateValue(key, value string, updatedBy int) error {
	// Update policies carry cross-field rules and an audit trail, so they are
	// writable only through the app-update endpoints. The comparison is
	// case-insensitive because AS_KEY uses a case-insensitive collation: a
	// differently cased key would otherwise reach the same row.
	if strings.HasPrefix(strings.ToLower(key), model.AppUpdatePolicyKeyPrefix) {
		return ErrAppSettingReadOnly
	}
	if key == "" || strings.TrimSpace(key) != key || utf8.RuneCountInString(key) > maxAppSettingKeyRunes {
		return ErrInvalidAppSettingKey
	}
	if !utf8.ValidString(value) || len(value) > maxAppSettingValueBytes {
		return ErrInvalidAppSettingValue
	}

	exists, err := s.store.UpdateValue(key, value, updatedBy)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAppSettingNotFound
	}
	if s.cache != nil {
		s.cache.Delete(appSettingsPublicCacheKey)
	}
	return nil
}

// InvalidatePublicSettingsCache drops the cached public payload so a write made
// through another service is visible to clients immediately.
func (s *AppSettingService) InvalidatePublicSettingsCache() {
	if s.cache != nil {
		s.cache.Delete(appSettingsPublicCacheKey)
	}
}

func clonePublicSettings(settings map[string]string) map[string]string {
	cloned := make(map[string]string, len(settings))
	for key, value := range settings {
		cloned[key] = value
	}
	return cloned
}
