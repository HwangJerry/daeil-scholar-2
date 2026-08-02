package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
)

type pushServicerStub struct {
	registerCalls   int
	unregisterCalls int
	getCalls        int
	updateCalls     int
	registration    model.PushDeviceRegistration
	deviceToken     string
	preferences     *model.PushPreferences
	updated         model.PushPreferences
	err             error
}

func (s *pushServicerStub) RegisterDevice(_ int, registration model.PushDeviceRegistration) error {
	s.registerCalls++
	s.registration = registration
	return s.err
}

func (s *pushServicerStub) UnregisterDevice(_ int, token string) error {
	s.unregisterCalls++
	s.deviceToken = token
	return s.err
}

func (s *pushServicerStub) GetPreferences(int) (*model.PushPreferences, error) {
	s.getCalls++
	return s.preferences, s.err
}

func (s *pushServicerStub) UpdatePreferences(_ int, preferences model.PushPreferences) (*model.PushPreferences, error) {
	s.updateCalls++
	s.updated = preferences
	return &preferences, s.err
}

func TestPushHandlerRegistersDeviceWithClosedCanonicalResponse(t *testing.T) {
	stub := &pushServicerStub{}
	h := &PushHandler{service: stub}
	request := authenticatedPushRequest(http.MethodPost, "/api/push/device/register", `{"platform":"android","deviceToken":"token-123","locale":"ko-KR"}`)
	recorder := httptest.NewRecorder()
	h.RegisterDevice(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "{\"status\":\"registered\"}\n" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if stub.registerCalls != 1 || stub.registration.DeviceToken != "token-123" {
		t.Fatalf("registration = %#v, calls = %d", stub.registration, stub.registerCalls)
	}
}

func TestPushHandlerRejectsUnknownRegistrationProperty(t *testing.T) {
	stub := &pushServicerStub{}
	h := &PushHandler{service: stub}
	request := authenticatedPushRequest(http.MethodPost, "/api/push/device/register", `{"platform":"android","deviceToken":"token-123","locale":"ko-KR","extra":true}`)
	recorder := httptest.NewRecorder()
	h.RegisterDevice(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
	}
	if stub.registerCalls != 0 {
		t.Fatalf("invalid body reached service %d times", stub.registerCalls)
	}
}

func TestPushHandlerUnregistersDeviceIdempotently(t *testing.T) {
	stub := &pushServicerStub{}
	h := &PushHandler{service: stub}
	request := authenticatedPushRequest(http.MethodPost, "/api/push/device/unregister", `{"deviceToken":"token-123"}`)
	recorder := httptest.NewRecorder()
	h.UnregisterDevice(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "{\"status\":\"unregistered\"}\n" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if stub.unregisterCalls != 1 || stub.deviceToken != "token-123" {
		t.Fatalf("token = %q, calls = %d", stub.deviceToken, stub.unregisterCalls)
	}
}

func TestPushHandlerRequiresAuthenticatedPrincipal(t *testing.T) {
	stub := &pushServicerStub{}
	h := &PushHandler{service: stub}
	request := httptest.NewRequest(http.MethodPost, "/api/push/device/register", bytes.NewBufferString(`{"platform":"android","deviceToken":"token-123","locale":"ko-KR"}`))
	recorder := httptest.NewRecorder()
	h.RegisterDevice(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if stub.registerCalls != 0 {
		t.Fatalf("unauthenticated request reached service %d times", stub.registerCalls)
	}
}

func TestPushHandlerGetsClosedCanonicalPreferences(t *testing.T) {
	stub := &pushServicerStub{preferences: &model.PushPreferences{MessageEnabled: true, MessagePreviewEnabled: false}}
	h := &PushHandler{service: stub}
	recorder := httptest.NewRecorder()
	h.GetPreferences(recorder, authenticatedPushRequest(http.MethodGet, "/api/push/preferences", ""))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "{\"messageEnabled\":true,\"messagePreviewEnabled\":false}\n" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if stub.getCalls != 1 {
		t.Fatalf("get calls = %d", stub.getCalls)
	}
}

func TestPushHandlerUpdatesPreferencesAndIgnoresOptionalNoticeEnabled(t *testing.T) {
	stub := &pushServicerStub{}
	h := &PushHandler{service: stub}
	body := `{"messageEnabled":false,"messagePreviewEnabled":true,"noticeEnabled":false}`
	recorder := httptest.NewRecorder()
	h.PutPreferences(recorder, authenticatedPushRequest(http.MethodPut, "/api/push/preferences", body))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "{\"messageEnabled\":false,\"messagePreviewEnabled\":true}\n" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if stub.updateCalls != 1 || stub.updated.MessageEnabled || !stub.updated.MessagePreviewEnabled {
		t.Fatalf("updated = %#v, calls = %d", stub.updated, stub.updateCalls)
	}
}

func TestPushHandlerRequiresBothCanonicalPreferenceBooleans(t *testing.T) {
	for _, body := range []string{
		`{"messageEnabled":true}`,
		`{"messagePreviewEnabled":true}`,
		`{"messageEnabled":true,"messagePreviewEnabled":true,"extra":false}`,
	} {
		stub := &pushServicerStub{}
		h := &PushHandler{service: stub}
		recorder := httptest.NewRecorder()
		h.PutPreferences(recorder, authenticatedPushRequest(http.MethodPut, "/api/push/preferences", body))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("body %s: status = %d, want 400", body, recorder.Code)
		}
		if stub.updateCalls != 0 {
			t.Errorf("body %s reached service", body)
		}
	}
}

func authenticatedPushRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	return request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42}))
}
