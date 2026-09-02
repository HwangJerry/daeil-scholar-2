// mobile_app_event_service.go — Validation and orchestration for mobile events.
package service

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dflh-saf/backend/internal/model"
)

var (
	ErrInvalidMobileEventBatch  = errors.New("invalid mobile event batch")
	ErrInvalidMobileEvent       = errors.New("invalid mobile event")
	ErrInvalidMobileEventFilter = errors.New("invalid mobile event filter")
)

const (
	maxMobileEventBatchSize   = 100
	maxMobileEventTypeRunes   = 50
	maxMobileAppVersionRunes  = 50
	maxMobileOSVersionRunes   = 50
	maxMobileDeviceModelRunes = 100
)

var allowedMobileEventTypes = map[string]struct{}{
	model.MobileEventSignupStart:    {},
	model.MobileEventSignupComplete: {},
	model.MobileEventApplyComplete:  {},
}

type MobileAppEventStore interface {
	InsertBatch(events []model.MobileAppEvent) error
	GetSummary(filter model.MobileAppEventSummaryFilter) ([]model.MobileAppEventSummary, error)
}

type MobileAppEventService struct {
	store MobileAppEventStore
	now   func() time.Time
}

func NewMobileAppEventService(store MobileAppEventStore) *MobileAppEventService {
	return &MobileAppEventService{store: store, now: time.Now}
}

func (s *MobileAppEventService) RecordBatch(events []model.MobileAppEvent, authenticatedUserID *int) error {
	if len(events) == 0 || len(events) > maxMobileEventBatchSize {
		return ErrInvalidMobileEventBatch
	}

	receivedAt := s.now().In(kstZone)
	for index := range events {
		if !validMobileAppEvent(events[index]) {
			return ErrInvalidMobileEvent
		}
		events[index].UserID = authenticatedUserID
		events[index].OccurredAt = events[index].OccurredAt.In(kstZone)
		events[index].CreatedAt = receivedAt
	}
	return s.store.InsertBatch(events)
}

func (s *MobileAppEventService) Summary(from, to time.Time, platform, eventType string) ([]model.MobileAppEventSummary, error) {
	if platform != "" && platform != model.MobilePlatformIOS && platform != model.MobilePlatformAndroid {
		return nil, ErrInvalidMobileEventFilter
	}
	if eventType != "" {
		if _, allowed := allowedMobileEventTypes[eventType]; !allowed {
			return nil, ErrInvalidMobileEventFilter
		}
	}
	return s.store.GetSummary(model.MobileAppEventSummaryFilter{
		From:        from,
		ToExclusive: to.AddDate(0, 0, 1),
		Platform:    platform,
		EventType:   eventType,
	})
}

func validMobileAppEvent(event model.MobileAppEvent) bool {
	if event.Platform != model.MobilePlatformIOS && event.Platform != model.MobilePlatformAndroid {
		return false
	}
	if _, allowed := allowedMobileEventTypes[event.EventType]; !allowed {
		return false
	}
	if event.EventType == "" || utf8.RuneCountInString(event.EventType) > maxMobileEventTypeRunes {
		return false
	}
	if !validRequiredMobileVersion(event.AppVersion, maxMobileAppVersionRunes) ||
		!validRequiredMobileVersion(event.OSVersion, maxMobileOSVersionRunes) {
		return false
	}
	if event.DeviceModel != nil {
		deviceModel := *event.DeviceModel
		if strings.TrimSpace(deviceModel) != deviceModel || deviceModel == "" || utf8.RuneCountInString(deviceModel) > maxMobileDeviceModelRunes {
			return false
		}
	}
	return !event.OccurredAt.IsZero()
}

func validRequiredMobileVersion(value string, maxRunes int) bool {
	return value != "" && strings.TrimSpace(value) == value && utf8.RuneCountInString(value) <= maxRunes
}
