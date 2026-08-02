package service

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/dflh-saf/backend/internal/model"
)

var (
	ErrInvalidPushRequest = errors.New("invalid push request")
)

type PushStore interface {
	RegisterDevice(usrSeq int, registration model.PushDeviceRegistration) error
	UnregisterDevice(usrSeq int, deviceToken string) error
	GetPreferences(usrSeq int) (*model.PushPreferences, error)
	UpsertPreferences(usrSeq int, preferences model.PushPreferences) error
}

type PushService struct {
	store PushStore
}

func NewPushService(store PushStore) *PushService {
	return &PushService{store: store}
}

func (s *PushService) RegisterDevice(usrSeq int, registration model.PushDeviceRegistration) error {
	if usrSeq <= 0 || !validPushDeviceRegistration(registration) {
		return ErrInvalidPushRequest
	}
	return s.store.RegisterDevice(usrSeq, registration)
}

func (s *PushService) UnregisterDevice(usrSeq int, deviceToken string) error {
	if usrSeq <= 0 || !validPushDeviceToken(deviceToken) {
		return ErrInvalidPushRequest
	}
	return s.store.UnregisterDevice(usrSeq, deviceToken)
}

func (s *PushService) GetPreferences(usrSeq int) (*model.PushPreferences, error) {
	if usrSeq <= 0 {
		return nil, ErrInvalidPushRequest
	}
	preferences, err := s.store.GetPreferences(usrSeq)
	if err != nil {
		return nil, err
	}
	if preferences == nil {
		return &model.PushPreferences{MessageEnabled: true, MessagePreviewEnabled: true}, nil
	}
	return preferences, nil
}

func (s *PushService) UpdatePreferences(usrSeq int, preferences model.PushPreferences) (*model.PushPreferences, error) {
	if usrSeq <= 0 {
		return nil, ErrInvalidPushRequest
	}
	if err := s.store.UpsertPreferences(usrSeq, preferences); err != nil {
		return nil, err
	}
	return &preferences, nil
}

func validPushDeviceRegistration(registration model.PushDeviceRegistration) bool {
	if !validPushDeviceToken(registration.DeviceToken) || !validPushLocale(registration.Locale) {
		return false
	}
	switch registration.Platform {
	case "android":
		return registration.APNSEnvironment == nil && registration.BundleID == nil
	case "ios":
		if registration.APNSEnvironment == nil || registration.BundleID == nil {
			return false
		}
		if *registration.APNSEnvironment != "sandbox" && *registration.APNSEnvironment != "production" {
			return false
		}
		return validPushBundleID(*registration.BundleID)
	default:
		return false
	}
}

func validPushDeviceToken(token string) bool {
	if len(token) == 0 || len(token) > 512 {
		return false
	}
	for i := 0; i < len(token); i++ {
		if token[i] < 0x21 || token[i] > 0x7e {
			return false
		}
	}
	return true
}

func validPushLocale(locale string) bool {
	if len(locale) == 0 || len(locale) > 20 {
		return false
	}
	for _, char := range locale {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

func validPushBundleID(bundleID string) bool {
	if strings.TrimSpace(bundleID) != bundleID || bundleID == "" || utf8.RuneCountInString(bundleID) > 255 {
		return false
	}
	for _, char := range bundleID {
		if char <= 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}
