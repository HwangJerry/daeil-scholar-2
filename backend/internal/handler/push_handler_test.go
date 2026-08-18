package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
)

type fakePushDeviceService struct {
	registerUsrSeq   int
	registerReq      model.PushDeviceRegistrationRequest
	unregisterUsrSeq int
	unregisterToken  string
}

func (f *fakePushDeviceService) RegisterDeviceToken(usrSeq int, req model.PushDeviceRegistrationRequest) error {
	f.registerUsrSeq = usrSeq
	f.registerReq = req
	return nil
}

func (f *fakePushDeviceService) UnregisterDeviceToken(usrSeq int, token string) error {
	f.unregisterUsrSeq = usrSeq
	f.unregisterToken = token
	return nil
}

func TestPushHandlerRegisterDeviceContract(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{"platform":"ios","deviceToken":" token-1 ","apnsEnvironment":"sandbox","bundleId":"kr.dflh.saf","locale":"ko-KR"}`)
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
		service.registerReq.APNsEnvironment != "sandbox" || service.registerReq.BundleID != "kr.dflh.saf" {
		t.Fatalf("unexpected register req: %#v", service.registerReq)
	}
}

func TestPushHandlerRegisterDeviceAcceptsAndroid(t *testing.T) {
	service := &fakePushDeviceService{}
	handler := NewPushHandler(service)
	body := bytes.NewBufferString(`{"platform":" ANDROID ","deviceToken":" fcm-token ","locale":"ko-KR"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/push/device/register", body)
	req = req.WithContext(middleware.SetAuthUser(req.Context(), &model.AuthUser{USRSeq: 42}))
	rec := httptest.NewRecorder()

	handler.RegisterDevice(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if service.registerReq.Platform != "android" || service.registerReq.DeviceToken != "fcm-token" ||
		service.registerReq.Locale != "ko-KR" {
		t.Fatalf("unexpected android register req: %#v", service.registerReq)
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
