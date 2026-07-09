package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	pushservice "github.com/dflh-saf/backend/internal/service"
)

type fakePushDeviceService struct {
	registerCalls    int
	registerUsrSeq   int
	registerReq      model.PushDeviceRegistrationRequest
	unregisterUsrSeq int
	unregisterToken  string
	preferences      model.PushPreferences
	updateUsrSeq     int
	updateReq        model.PushPreferencesUpdateRequest
}

func (f *fakePushDeviceService) RegisterDeviceToken(usrSeq int, req model.PushDeviceRegistrationRequest) error {
	f.registerCalls++
	f.registerUsrSeq = usrSeq
	f.registerReq = req
	return nil
}

func (f *fakePushDeviceService) UnregisterDeviceToken(usrSeq int, token string) error {
	f.unregisterUsrSeq = usrSeq
	f.unregisterToken = token
	return nil
}

func (f *fakePushDeviceService) GetPreferences(int) (model.PushPreferences, error) {
	if f.preferences == (model.PushPreferences{}) {
		return model.DefaultPushPreferences(), nil
	}
	return f.preferences, nil
}

func (f *fakePushDeviceService) UpdatePreferences(usrSeq int, req model.PushPreferencesUpdateRequest) (model.PushPreferences, error) {
	f.updateUsrSeq = usrSeq
	f.updateReq = req
	preferences, ok := req.Preferences()
	if !ok {
		return model.PushPreferences{}, pushservice.ErrInvalidPushPreferences
	}
	f.preferences = preferences
	return f.preferences, nil
}

func TestPushHandlerRegisterDeviceContract(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{"platform":"ios","deviceToken":" token-1 ","apnsEnvironment":"sandbox","bundleId":"com.daeil.dflhsafv2","locale":"ko-KR"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/push/device/register", body)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.RegisterDevice(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if service.registerUsrSeq != 42 {
		t.Fatalf("expected usrSeq 42, got %d", service.registerUsrSeq)
	}
	if service.registerReq.Platform != "ios" || service.registerReq.DeviceToken != "token-1" ||
		service.registerReq.APNsEnvironment != "sandbox" || service.registerReq.BundleID != "com.daeil.dflhsafv2" {
		t.Fatalf("unexpected register req: %#v", service.registerReq)
	}
}

func TestPushHandlerRegisterDeviceAcceptsAndroid(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{"platform":" ANDROID ","deviceToken":" fcm-token ","apnsEnvironment":"sandbox","bundleId":"com.daeil.dflhsafv2","locale":"ko-KR"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/push/device/register", body)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.RegisterDevice(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if service.registerCalls != 1 {
		t.Fatalf("expected one service call, got %d", service.registerCalls)
	}
	if service.registerReq.Platform != "android" || service.registerReq.DeviceToken != "fcm-token" ||
		service.registerReq.APNsEnvironment != "" || service.registerReq.BundleID != "" ||
		service.registerReq.Locale != "ko-KR" {
		t.Fatalf("unexpected android register req: %#v", service.registerReq)
	}
}

func TestPushHandlerRegisterDeviceAcceptsMaxLengthToken(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	token := strings.Repeat("a", maxPushDeviceTokenLength)
	body, err := json.Marshal(map[string]string{
		"platform":    "android",
		"deviceToken": token,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/push/device/register", bytes.NewReader(body))
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.RegisterDevice(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if service.registerCalls != 1 {
		t.Fatalf("expected one service call, got %d", service.registerCalls)
	}
	if service.registerReq.DeviceToken != token {
		t.Fatalf("unexpected token length: got %d want %d", len(service.registerReq.DeviceToken), maxPushDeviceTokenLength)
	}
}

func TestPushHandlerRegisterDeviceRejectsOversizedToken(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	token := strings.Repeat("a", maxPushDeviceTokenLength+1)
	body, err := json.Marshal(map[string]string{
		"platform":    "android",
		"deviceToken": token,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/push/device/register", bytes.NewReader(body))
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.RegisterDevice(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "INVALID_TOKEN")
	if service.registerCalls != 0 {
		t.Fatalf("expected service not called, got %d calls", service.registerCalls)
	}
}

func TestPushHandlerRegisterDeviceRejectsUnknownPlatform(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{"platform":"web","deviceToken":"token-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/push/device/register", body)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.RegisterDevice(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if service.registerReq.DeviceToken != "" {
		t.Fatalf("expected service not called, got %#v", service.registerReq)
	}
}

func TestPushHandlerRegisterDeviceRejectsMissingAPNsEnvironment(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{"platform":"ios","deviceToken":"token-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/push/device/register", body)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.RegisterDevice(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "INVALID_APNS_ENVIRONMENT")
	if service.registerCalls != 0 {
		t.Fatalf("expected service not called, got %d calls", service.registerCalls)
	}
}

func TestPushHandlerRegisterDeviceRejectsInvalidAPNsEnvironment(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{"platform":"ios","deviceToken":"token-1","apnsEnvironment":"staging"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/push/device/register", body)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.RegisterDevice(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "INVALID_APNS_ENVIRONMENT")
	if service.registerCalls != 0 {
		t.Fatalf("expected service not called, got %d calls", service.registerCalls)
	}
}

func TestPushHandlerRegisterDeviceNormalizesDebugAlias(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{"platform":"ios","deviceToken":"token-1","environment":"debug"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/push/device/register", body)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.RegisterDevice(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if service.registerReq.APNsEnvironment != "sandbox" {
		t.Fatalf("expected sandbox environment, got %#v", service.registerReq)
	}
}

func TestPushHandlerRegisterDeviceNormalizesTestFlightAlias(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{"platform":"ios","deviceToken":"token-1","environment":"testflight"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/push/device/register", body)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.RegisterDevice(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if service.registerReq.APNsEnvironment != "production" {
		t.Fatalf("expected production environment, got %#v", service.registerReq)
	}
}

func TestPushHandlerUnregisterDeviceContract(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{"deviceToken":" token-1 "}`)
	req := httptest.NewRequest(http.MethodPost, "/api/push/device/unregister", body)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.UnregisterDevice(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if service.unregisterUsrSeq != 42 || service.unregisterToken != "token-1" {
		t.Fatalf("unexpected unregister call: usrSeq=%d token=%q", service.unregisterUsrSeq, service.unregisterToken)
	}
}

func TestPushHandlerGetPreferencesContract(t *testing.T) {
	service := &fakePushDeviceService{
		preferences: model.PushPreferences{
			NoticeEnabled:  false,
			MessageEnabled: true,
		},
	}
	handler := NewPushHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/push/preferences", nil)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.GetPreferences(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var got model.PushPreferences
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode preferences: %v", err)
	}
	if got.NoticeEnabled || !got.MessageEnabled {
		t.Fatalf("unexpected preferences: %#v", got)
	}
}

func TestPushHandlerUpdatePreferencesContract(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{"noticeEnabled":false,"messageEnabled":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/push/preferences", body)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.UpdatePreferences(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if service.updateUsrSeq != 42 ||
		service.updateReq.NoticeEnabled == nil ||
		service.updateReq.MessageEnabled == nil ||
		*service.updateReq.NoticeEnabled ||
		!*service.updateReq.MessageEnabled {
		t.Fatalf("unexpected update call: usrSeq=%d req=%#v", service.updateUsrSeq, service.updateReq)
	}
	var got model.PushPreferences
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode preferences: %v", err)
	}
	if got.NoticeEnabled || !got.MessageEnabled {
		t.Fatalf("unexpected preferences response: %#v", got)
	}
}

func TestPushHandlerUpdatePreferencesRejectsMissingFields(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPut, "/api/push/preferences", body)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.UpdatePreferences(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "INVALID_PUSH_PREFERENCES")
}

func assertAPIErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var body model.APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v body=%s", err, rec.Body.String())
	}
	if body.Code != want {
		t.Fatalf("unexpected error code: got %q want %q body=%s", body.Code, want, rec.Body.String())
	}
}
