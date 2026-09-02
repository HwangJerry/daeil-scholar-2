// app_setting_service_test.go — Cache and update behavior tests for application settings.
package service

import (
	"errors"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
)

type appSettingStoreStub struct {
	all             []model.AppSetting
	public          []model.AppSetting
	publicListCalls int
	updateExists    bool
	updateErr       error
}

func (s *appSettingStoreStub) ListAll() ([]model.AppSetting, error) {
	return s.all, nil
}

func (s *appSettingStoreStub) ListPublic() ([]model.AppSetting, error) {
	s.publicListCalls++
	return s.public, nil
}

func (s *appSettingStoreStub) UpdateValue(key, value string, _ int) (bool, error) {
	if s.updateErr != nil {
		return false, s.updateErr
	}
	if s.updateExists {
		for index := range s.public {
			if s.public[index].Key == key {
				s.public[index].Value = value
			}
		}
	}
	return s.updateExists, nil
}

func TestAppSettingServiceCachesPublicSettingsAndInvalidatesAfterUpdate(t *testing.T) {
	store := &appSettingStoreStub{
		public:       []model.AppSetting{{Key: "kakao_open_chat_url", Value: "first", Public: model.AppSettingPublic}},
		updateExists: true,
	}
	service := NewAppSettingService(store, cache.New(time.Minute, time.Minute))

	first, err := service.GetPublicSettings()
	if err != nil {
		t.Fatal(err)
	}
	first["kakao_open_chat_url"] = "caller-mutation"
	second, err := service.GetPublicSettings()
	if err != nil {
		t.Fatal(err)
	}
	if second["kakao_open_chat_url"] != "first" || store.publicListCalls != 1 {
		t.Fatalf("cached settings = %#v, public list calls = %d", second, store.publicListCalls)
	}

	if err := service.UpdateValue("kakao_open_chat_url", "second", 7); err != nil {
		t.Fatalf("UpdateValue() error = %v", err)
	}
	refreshed, err := service.GetPublicSettings()
	if err != nil {
		t.Fatal(err)
	}
	if refreshed["kakao_open_chat_url"] != "second" || store.publicListCalls != 2 {
		t.Fatalf("refreshed settings = %#v, public list calls = %d", refreshed, store.publicListCalls)
	}
}

func TestAppSettingServiceReturnsNotFoundWithoutInvalidatingCache(t *testing.T) {
	store := &appSettingStoreStub{
		public:       []model.AppSetting{{Key: "known", Value: "cached"}},
		updateExists: false,
	}
	service := NewAppSettingService(store, cache.New(time.Minute, time.Minute))
	if _, err := service.GetPublicSettings(); err != nil {
		t.Fatal(err)
	}

	err := service.UpdateValue("missing", "value", 7)
	if !errors.Is(err, ErrAppSettingNotFound) {
		t.Fatalf("UpdateValue() error = %v, want ErrAppSettingNotFound", err)
	}
	if _, err := service.GetPublicSettings(); err != nil {
		t.Fatal(err)
	}
	if store.publicListCalls != 1 {
		t.Fatalf("public list calls = %d, want cache retained", store.publicListCalls)
	}
}

func TestAppSettingServiceValidatesKeyAndTextCapacity(t *testing.T) {
	service := NewAppSettingService(&appSettingStoreStub{}, nil)
	if err := service.UpdateValue(" bad-key", "value", 7); !errors.Is(err, ErrInvalidAppSettingKey) {
		t.Fatalf("invalid key error = %v", err)
	}
	oversized := make([]byte, maxAppSettingValueBytes+1)
	if err := service.UpdateValue("key", string(oversized), 7); !errors.Is(err, ErrInvalidAppSettingValue) {
		t.Fatalf("oversized value error = %v", err)
	}
}
