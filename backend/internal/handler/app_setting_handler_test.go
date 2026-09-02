// app_setting_handler_test.go — HTTP contract tests for application settings endpoints.
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type appSettingServiceStub struct {
	settings  []model.AppSetting
	public    map[string]string
	updateErr error
	key       string
	value     string
	updatedBy int
}

func (s *appSettingServiceStub) ListSettings() ([]model.AppSetting, error) {
	return s.settings, nil
}

func (s *appSettingServiceStub) GetPublicSettings() (map[string]string, error) {
	return s.public, nil
}

func (s *appSettingServiceStub) UpdateValue(key, value string, updatedBy int) error {
	s.key = key
	s.value = value
	s.updatedBy = updatedBy
	return s.updateErr
}

func TestAppSettingHandlerPublicReturnsKeyValueObject(t *testing.T) {
	stub := &appSettingServiceStub{public: map[string]string{
		"kakao_open_chat_url": "https://open.kakao.com/o/gNLYTuui",
	}}
	handler := NewAppSettingHandler(stub)
	response := httptest.NewRecorder()

	handler.Public(response, httptest.NewRequest(http.MethodGet, "/api/settings/public", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["kakao_open_chat_url"] != "https://open.kakao.com/o/gNLYTuui" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestAppSettingHandlerUpdatePassesAuthenticatedOperator(t *testing.T) {
	stub := &appSettingServiceStub{}
	handler := NewAppSettingHandler(stub)
	router := chi.NewRouter()
	router.Put("/api/admin/settings/{key}", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(middleware.SetAuthUser(r.Context(), &model.AuthUser{USRSeq: 7}))
		handler.Update(w, r)
	})
	request := httptest.NewRequest(http.MethodPut, "/api/admin/settings/kakao_open_chat_url", strings.NewReader(`{"value":"https://example.com/new"}`))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if stub.key != "kakao_open_chat_url" || stub.value != "https://example.com/new" || stub.updatedBy != 7 {
		t.Fatalf("update = key %q, value %q, operator %d", stub.key, stub.value, stub.updatedBy)
	}
}

func TestAppSettingHandlerUpdateRejectsMissingValue(t *testing.T) {
	handler := NewAppSettingHandler(&appSettingServiceStub{})
	request := httptest.NewRequest(http.MethodPut, "/api/admin/settings/key", strings.NewReader(`{}`))
	response := httptest.NewRecorder()

	handler.Update(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_BODY"`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestAppSettingHandlerUpdateReturnsNotFound(t *testing.T) {
	stub := &appSettingServiceStub{updateErr: service.ErrAppSettingNotFound}
	handler := NewAppSettingHandler(stub)
	router := chi.NewRouter()
	router.Put("/api/admin/settings/{key}", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(middleware.SetAuthUser(r.Context(), &model.AuthUser{USRSeq: 7}))
		handler.Update(w, r)
	})
	request := httptest.NewRequest(http.MethodPut, "/api/admin/settings/missing", strings.NewReader(`{"value":"new"}`))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), `"code":"SETTING_NOT_FOUND"`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}
