package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

type pushStoreStub struct {
	registerCalls   int
	unregisterCalls int
	getCalls        int
	upsertCalls     int
	registration    model.PushDeviceRegistration
	unregisterSeq   int
	unregisterToken string
	preferences     *model.PushPreferences
	upserted        model.PushPreferences
	err             error
}

func (s *pushStoreStub) RegisterDevice(_ int, registration model.PushDeviceRegistration) error {
	s.registerCalls++
	s.registration = registration
	return s.err
}

func (s *pushStoreStub) UnregisterDevice(usrSeq int, token string) error {
	s.unregisterCalls++
	s.unregisterSeq = usrSeq
	s.unregisterToken = token
	return s.err
}

func (s *pushStoreStub) GetPreferences(int) (*model.PushPreferences, error) {
	s.getCalls++
	return s.preferences, s.err
}

func (s *pushStoreStub) UpsertPreferences(_ int, preferences model.PushPreferences) error {
	s.upsertCalls++
	s.upserted = preferences
	return s.err
}

func TestPushServiceRegistersCanonicalAndroidDevice(t *testing.T) {
	store := &pushStoreStub{}
	svc := NewPushService(store)
	request := model.PushDeviceRegistration{Platform: "android", DeviceToken: "token-123", Locale: "ko-KR"}

	if err := svc.RegisterDevice(42, request); err != nil {
		t.Fatalf("RegisterDevice error = %v", err)
	}
	if store.registerCalls != 1 || store.registration != request {
		t.Fatalf("registration = %#v, calls = %d", store.registration, store.registerCalls)
	}
}

func TestPushServiceRegistersCanonicalIOSDevice(t *testing.T) {
	store := &pushStoreStub{}
	svc := NewPushService(store)
	environment, bundleID := "sandbox", "com.daeil.dflhsafv2"
	request := model.PushDeviceRegistration{
		Platform: "ios", DeviceToken: "abcdef012345", Locale: "ko_KR",
		APNSEnvironment: &environment, BundleID: &bundleID,
	}

	if err := svc.RegisterDevice(42, request); err != nil {
		t.Fatalf("RegisterDevice error = %v", err)
	}
	if store.registerCalls != 1 {
		t.Fatalf("register calls = %d, want 1", store.registerCalls)
	}
}

func TestPushServiceRejectsInvalidDeviceRegistrationBeforeStore(t *testing.T) {
	environment, bundleID := "sandbox", "com.example.app"
	longToken := strings.Repeat("a", 513)
	tests := []model.PushDeviceRegistration{
		{Platform: "", DeviceToken: "token", Locale: "ko-KR"},
		{Platform: "web", DeviceToken: "token", Locale: "ko-KR"},
		{Platform: "android", DeviceToken: "", Locale: "ko-KR"},
		{Platform: "android", DeviceToken: "token with space", Locale: "ko-KR"},
		{Platform: "android", DeviceToken: "토큰", Locale: "ko-KR"},
		{Platform: "android", DeviceToken: longToken, Locale: "ko-KR"},
		{Platform: "android", DeviceToken: "token", Locale: ""},
		{Platform: "android", DeviceToken: "token", Locale: strings.Repeat("k", 21)},
		{Platform: "android", DeviceToken: "token", Locale: "ko-KR", APNSEnvironment: &environment},
		{Platform: "android", DeviceToken: "token", Locale: "ko-KR", BundleID: &bundleID},
		{Platform: "ios", DeviceToken: "token", Locale: "ko-KR"},
		{Platform: "ios", DeviceToken: "token", Locale: "ko-KR", APNSEnvironment: &environment},
		{Platform: "ios", DeviceToken: "token", Locale: "ko-KR", BundleID: &bundleID},
	}

	for i, request := range tests {
		store := &pushStoreStub{}
		err := NewPushService(store).RegisterDevice(42, request)
		if !errors.Is(err, ErrInvalidPushRequest) {
			t.Errorf("case %d error = %v, want ErrInvalidPushRequest", i, err)
		}
		if store.registerCalls != 0 {
			t.Errorf("case %d register calls = %d, want 0", i, store.registerCalls)
		}
	}
}

func TestPushServiceUnregistersOnlyValidatedToken(t *testing.T) {
	store := &pushStoreStub{}
	svc := NewPushService(store)
	if err := svc.UnregisterDevice(42, "token-123"); err != nil {
		t.Fatalf("UnregisterDevice error = %v", err)
	}
	if store.unregisterCalls != 1 || store.unregisterSeq != 42 || store.unregisterToken != "token-123" {
		t.Fatalf("unregister = (%d, %q), calls = %d", store.unregisterSeq, store.unregisterToken, store.unregisterCalls)
	}

	invalidStore := &pushStoreStub{}
	if err := NewPushService(invalidStore).UnregisterDevice(42, "bad token"); !errors.Is(err, ErrInvalidPushRequest) {
		t.Fatalf("invalid token error = %v", err)
	}
	if invalidStore.unregisterCalls != 0 {
		t.Fatalf("invalid token reached store %d times", invalidStore.unregisterCalls)
	}
}

func TestPushServiceReturnsDefaultPreferencesWithoutWritingMissingRow(t *testing.T) {
	store := &pushStoreStub{}
	preferences, err := NewPushService(store).GetPreferences(42)
	if err != nil {
		t.Fatalf("GetPreferences error = %v", err)
	}
	if preferences == nil || !preferences.MessageEnabled || !preferences.MessagePreviewEnabled {
		t.Fatalf("preferences = %#v, want true/true", preferences)
	}
	if store.getCalls != 1 || store.upsertCalls != 0 {
		t.Fatalf("get calls = %d, upsert calls = %d", store.getCalls, store.upsertCalls)
	}
}

func TestPushServiceReturnsStoredPreferences(t *testing.T) {
	stored := &model.PushPreferences{MessageEnabled: true, MessagePreviewEnabled: false}
	store := &pushStoreStub{preferences: stored}
	preferences, err := NewPushService(store).GetPreferences(42)
	if err != nil {
		t.Fatalf("GetPreferences error = %v", err)
	}
	if preferences != stored {
		t.Fatalf("preferences = %#v, want stored pointer", preferences)
	}
}

func TestPushServiceUpsertsAndReturnsCanonicalPreferences(t *testing.T) {
	store := &pushStoreStub{}
	request := model.PushPreferences{MessageEnabled: false, MessagePreviewEnabled: true}
	preferences, err := NewPushService(store).UpdatePreferences(42, request)
	if err != nil {
		t.Fatalf("UpdatePreferences error = %v", err)
	}
	if store.upsertCalls != 1 || store.upserted != request {
		t.Fatalf("upserted = %#v, calls = %d", store.upserted, store.upsertCalls)
	}
	if preferences == nil || *preferences != request {
		t.Fatalf("preferences = %#v", preferences)
	}
}
