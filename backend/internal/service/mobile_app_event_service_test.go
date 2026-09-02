package service

import (
	"errors"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

type mobileAppEventStoreStub struct {
	events      []model.MobileAppEvent
	from        time.Time
	toExclusive time.Time
	summary     []model.MobileAppEventSummary
	err         error
}

func (s *mobileAppEventStoreStub) InsertBatch(events []model.MobileAppEvent) error {
	s.events = events
	return s.err
}

func (s *mobileAppEventStoreStub) GetSummary(filter model.MobileAppEventSummaryFilter) ([]model.MobileAppEventSummary, error) {
	s.from = filter.From
	s.toExclusive = filter.ToExclusive
	return s.summary, s.err
}

func TestMobileAppEventServiceValidatesAndEnrichesBatch(t *testing.T) {
	store := &mobileAppEventStoreStub{}
	receivedAt := time.Date(2026, 9, 2, 3, 0, 0, 0, time.UTC)
	service := NewMobileAppEventService(store)
	service.now = func() time.Time { return receivedAt }
	userID := 42
	deviceModel := "Pixel 10"
	event := model.MobileAppEvent{
		Platform: model.MobilePlatformAndroid, EventType: model.MobileEventApplyComplete,
		AppVersion: "4.2.1", OSVersion: "16", DeviceModel: &deviceModel,
		OccurredAt: time.Date(2026, 9, 2, 2, 30, 0, 0, time.UTC),
	}

	if err := service.RecordBatch([]model.MobileAppEvent{event}, &userID); err != nil {
		t.Fatal(err)
	}
	if len(store.events) != 1 || store.events[0].UserID == nil || *store.events[0].UserID != userID {
		t.Fatalf("events = %#v", store.events)
	}
	if store.events[0].OccurredAt.Hour() != 11 || store.events[0].CreatedAt.Hour() != 12 {
		t.Fatalf("event timestamps were not normalized to KST: %#v", store.events[0])
	}
}

func TestMobileAppEventServiceRejectsNonWhitelistedValues(t *testing.T) {
	valid := model.MobileAppEvent{
		Platform: model.MobilePlatformIOS, EventType: model.MobileEventSignupStart,
		AppVersion: "1.0.0", OSVersion: "18.0", OccurredAt: time.Now(),
	}
	tests := []struct {
		name   string
		mutate func(*model.MobileAppEvent)
	}{
		{name: "platform", mutate: func(event *model.MobileAppEvent) { event.Platform = "web" }},
		{name: "event type", mutate: func(event *model.MobileAppEvent) { event.EventType = "arbitrary_event" }},
		{name: "app version", mutate: func(event *model.MobileAppEvent) { event.AppVersion = "" }},
		{name: "occurred at", mutate: func(event *model.MobileAppEvent) { event.OccurredAt = time.Time{} }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := valid
			test.mutate(&event)
			service := NewMobileAppEventService(&mobileAppEventStoreStub{})
			if err := service.RecordBatch([]model.MobileAppEvent{event}, nil); !errors.Is(err, ErrInvalidMobileEvent) {
				t.Fatalf("error = %v, want ErrInvalidMobileEvent", err)
			}
		})
	}
}

func TestMobileAppEventServiceRejectsEmptyAndOversizedBatches(t *testing.T) {
	service := NewMobileAppEventService(&mobileAppEventStoreStub{})
	if err := service.RecordBatch(nil, nil); !errors.Is(err, ErrInvalidMobileEventBatch) {
		t.Fatalf("empty batch error = %v", err)
	}
	if err := service.RecordBatch(make([]model.MobileAppEvent, maxMobileEventBatchSize+1), nil); !errors.Is(err, ErrInvalidMobileEventBatch) {
		t.Fatalf("oversized batch error = %v", err)
	}
}

func TestMobileAppEventServiceUsesInclusiveSummaryEndDate(t *testing.T) {
	store := &mobileAppEventStoreStub{}
	service := NewMobileAppEventService(store)
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, kstZone)
	to := time.Date(2026, 8, 31, 0, 0, 0, 0, kstZone)

	if _, err := service.Summary(from, to, model.MobilePlatformIOS, model.MobileEventSignupStart); err != nil {
		t.Fatal(err)
	}
	if !store.from.Equal(from) || !store.toExclusive.Equal(to.AddDate(0, 0, 1)) {
		t.Fatalf("range = [%s, %s)", store.from, store.toExclusive)
	}
}

func TestMobileAppEventServiceRejectsInvalidSummaryFilters(t *testing.T) {
	service := NewMobileAppEventService(&mobileAppEventStoreStub{})
	today := time.Now()
	for _, filter := range []struct {
		platform  string
		eventType string
	}{
		{platform: "web"},
		{eventType: "arbitrary_event"},
	} {
		if _, err := service.Summary(today, today, filter.platform, filter.eventType); !errors.Is(err, ErrInvalidMobileEventFilter) {
			t.Fatalf("filter = %#v, error = %v", filter, err)
		}
	}
}
