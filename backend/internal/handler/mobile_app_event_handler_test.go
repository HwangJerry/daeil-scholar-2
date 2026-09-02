package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

type mobileAppEventServicerStub struct {
	events    []model.MobileAppEvent
	userID    *int
	from      time.Time
	to        time.Time
	platform  string
	eventType string
	items     []model.MobileAppEventSummary
	err       error
}

func (s *mobileAppEventServicerStub) RecordBatch(events []model.MobileAppEvent, userID *int) error {
	s.events = events
	s.userID = userID
	return s.err
}

func (s *mobileAppEventServicerStub) Summary(from, to time.Time, platform, eventType string) ([]model.MobileAppEventSummary, error) {
	s.from = from
	s.to = to
	s.platform = platform
	s.eventType = eventType
	return s.items, s.err
}

func TestMobileAppEventHandlerCollectsAnonymousBatch(t *testing.T) {
	stub := &mobileAppEventServicerStub{}
	handler := &MobileAppEventHandler{service: stub}
	body := `{"events":[{"platform":"ios","eventType":"signup_start","appVersion":"2.3.0","osVersion":"18.6","deviceModel":"iPhone17,1","occurredAt":"2026-09-02T12:30:00+09:00"}]}`
	response := httptest.NewRecorder()

	handler.Collect(response, httptest.NewRequest(http.MethodPost, "/api/mobile/events", strings.NewReader(body)))

	if response.Code != http.StatusAccepted || response.Body.String() != "{\"accepted\":1}\n" {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if len(stub.events) != 1 || stub.userID != nil || stub.events[0].EventType != model.MobileEventSignupStart {
		t.Fatalf("events = %#v, userID = %#v", stub.events, stub.userID)
	}
}

func TestMobileAppEventHandlerUsesAuthenticatedUser(t *testing.T) {
	stub := &mobileAppEventServicerStub{}
	handler := &MobileAppEventHandler{service: stub}
	body := `{"events":[{"platform":"android","eventType":"signup_complete","appVersion":"3.0.0","osVersion":"16","occurredAt":"2026-09-02T03:30:00Z"}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/mobile/events", strings.NewReader(body))
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42}))
	response := httptest.NewRecorder()

	handler.Collect(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if stub.userID == nil || *stub.userID != 42 {
		t.Fatalf("userID = %#v", stub.userID)
	}
}

func TestMobileAppEventHandlerRejectsUnknownFields(t *testing.T) {
	stub := &mobileAppEventServicerStub{}
	handler := &MobileAppEventHandler{service: stub}
	body := `{"events":[],"userId":42}`
	response := httptest.NewRecorder()

	handler.Collect(response, httptest.NewRequest(http.MethodPost, "/api/mobile/events", bytes.NewBufferString(body)))

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_BODY"`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if stub.events != nil {
		t.Fatal("invalid request reached service")
	}
}

func TestMobileAppEventHandlerMapsValidationErrors(t *testing.T) {
	tests := []struct {
		err  error
		code string
	}{
		{err: service.ErrInvalidMobileEventBatch, code: "INVALID_BATCH"},
		{err: service.ErrInvalidMobileEvent, code: "INVALID_EVENT"},
		{err: errors.New("database unavailable"), code: "EVENT_STORE_FAILED"},
	}

	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			stub := &mobileAppEventServicerStub{err: test.err}
			handler := &MobileAppEventHandler{service: stub}
			body := `{"events":[{"platform":"ios","eventType":"signup_start","appVersion":"1","osVersion":"18","occurredAt":"2026-09-02T00:00:00Z"}]}`
			response := httptest.NewRecorder()
			handler.Collect(response, httptest.NewRequest(http.MethodPost, "/api/mobile/events", strings.NewReader(body)))
			if !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestMobileAppEventHandlerReturnsSummaryForInclusiveRange(t *testing.T) {
	stub := &mobileAppEventServicerStub{items: []model.MobileAppEventSummary{
		{Platform: "ios", EventType: "signup_start", Count: 12},
	}}
	handler := &MobileAppEventHandler{service: stub}
	response := httptest.NewRecorder()

	handler.Summary(response, httptest.NewRequest(http.MethodGet, "/api/admin/mobile-events/summary?from=2026-08-01&to=2026-08-31&platform=ios&event_type=signup_start", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{`"from":"2026-08-01"`, `"to":"2026-08-31"`, `"platform":"ios"`, `"eventType":"signup_start"`, `"count":12`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("body = %s, want %s", response.Body.String(), expected)
		}
	}
	if stub.from.Format(mobileEventDateLayout) != "2026-08-01" || stub.to.Format(mobileEventDateLayout) != "2026-08-31" {
		t.Fatalf("range = %s to %s", stub.from, stub.to)
	}
	if stub.platform != "ios" || stub.eventType != "signup_start" {
		t.Fatalf("filters = platform %q, event type %q", stub.platform, stub.eventType)
	}
}

func TestMobileAppEventHandlerRejectsInvalidSummaryRange(t *testing.T) {
	handler := &MobileAppEventHandler{service: &mobileAppEventServicerStub{}}
	for _, target := range []string{
		"/api/admin/mobile-events/summary?from=invalid",
		"/api/admin/mobile-events/summary?from=2026-09-02&to=2026-09-01",
	} {
		response := httptest.NewRecorder()
		handler.Summary(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("target = %s, status = %d, body = %s", target, response.Code, response.Body.String())
		}
	}
}

func TestMobileAppEventHandlerRejectsInvalidSummaryFilters(t *testing.T) {
	handler := &MobileAppEventHandler{service: &mobileAppEventServicerStub{err: service.ErrInvalidMobileEventFilter}}
	response := httptest.NewRecorder()
	handler.Summary(response, httptest.NewRequest(http.MethodGet, "/api/admin/mobile-events/summary?platform=web", nil))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_FILTER"`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}
