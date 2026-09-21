// admin_notification_template_handler_test.go — HTTP contract for notification text administration.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type notificationTemplateServiceStub struct {
	views       []service.TemplateView
	saved       *service.TemplateView
	listErr     error
	updateErr   error
	updateCalls int
	lastInput   service.UpdateNotificationTemplateInput
}

func (s *notificationTemplateServiceStub) List() ([]service.TemplateView, error) {
	return s.views, s.listErr
}

func (s *notificationTemplateServiceStub) Update(
	input service.UpdateNotificationTemplateInput,
) (*service.TemplateView, error) {
	s.updateCalls++
	s.lastInput = input
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	return s.saved, nil
}

const updateTemplateBody = `{"title":"동문 인증 완료","body":"인증이 완료되었습니다.","expectedVersion":3}`

// updateTemplateRouter mounts the endpoint with an authenticated operator so the
// tests exercise the same URL parameter binding production uses.
func updateTemplateRouter(handler *AdminNotificationTemplateHandler) chi.Router {
	router := chi.NewRouter()
	router.Put("/api/admin/notification-templates/{key}", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(middleware.SetAuthUser(r.Context(), &model.AuthUser{USRSeq: 11}))
		handler.Update(w, r)
	})
	return router
}

func anonymousUpdateTemplateRouter(handler *AdminNotificationTemplateHandler) chi.Router {
	router := chi.NewRouter()
	router.Put("/api/admin/notification-templates/{key}", handler.Update)
	return router
}

func updateTemplateRequest(body string) *http.Request {
	return httptest.NewRequest(
		http.MethodPut,
		"/api/admin/notification-templates/"+model.NotificationTemplateVerificationApproved,
		strings.NewReader(body),
	)
}

func decodeAPIError(t *testing.T, body []byte) model.APIError {
	t.Helper()
	var apiError model.APIError
	if err := json.Unmarshal(body, &apiError); err != nil {
		t.Fatal(err)
	}
	return apiError
}

func TestAdminNotificationTemplateHandlerListsEveryCatalogRow(t *testing.T) {
	templates := &notificationTemplateServiceStub{views: []service.TemplateView{
		{Key: model.NotificationTemplatePhoneVerificationSMS, Channel: model.NotificationChannelSMS},
		{Key: model.NotificationTemplateNoticeNew, Channel: model.NotificationChannelPush, Version: 2},
	}}
	handler := NewAdminNotificationTemplateHandler(templates)
	response := httptest.NewRecorder()

	handler.List(response, httptest.NewRequest(http.MethodGet, "/api/admin/notification-templates", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload []service.TemplateView
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 2 || payload[0].Key != model.NotificationTemplatePhoneVerificationSMS {
		t.Fatalf("payload = %#v", payload)
	}
}

// An empty catalog must still serialise as [] so the SPA can map over it.
func TestAdminNotificationTemplateHandlerListsAnEmptyArrayNotNull(t *testing.T) {
	handler := NewAdminNotificationTemplateHandler(&notificationTemplateServiceStub{})
	response := httptest.NewRecorder()

	handler.List(response, httptest.NewRequest(http.MethodGet, "/api/admin/notification-templates", nil))

	if body := strings.TrimSpace(response.Body.String()); body != "[]" {
		t.Fatalf("body = %s, want []", body)
	}
}

func TestAdminNotificationTemplateHandlerUpdateReturnsTheSavedRow(t *testing.T) {
	templates := &notificationTemplateServiceStub{
		saved: &service.TemplateView{
			Key:     model.NotificationTemplateVerificationApproved,
			Channel: model.NotificationChannelPush,
			Title:   "동문 인증 완료",
			Body:    "인증이 완료되었습니다.",
			Version: 4,
		},
	}
	handler := NewAdminNotificationTemplateHandler(templates)
	response := httptest.NewRecorder()

	updateTemplateRouter(handler).ServeHTTP(response, updateTemplateRequest(updateTemplateBody))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	input := templates.lastInput
	if input.Key != model.NotificationTemplateVerificationApproved || input.UpdatedBy != 11 {
		t.Fatalf("input key = %q, operator = %d", input.Key, input.UpdatedBy)
	}
	if input.Title != "동문 인증 완료" || input.Body != "인증이 완료되었습니다." || input.ExpectedVersion != 3 {
		t.Fatalf("input = %#v", input)
	}
	var view service.TemplateView
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Version != 4 || view.Key != model.NotificationTemplateVerificationApproved {
		t.Fatalf("view = %#v", view)
	}
}

// The write itself returns the saved row, so a second read can neither fail
// after a committed save nor hand back a rival administrator's edit.
func TestAdminNotificationTemplateHandlerDoesNotReReadAfterTheWrite(t *testing.T) {
	templates := &notificationTemplateServiceStub{saved: &service.TemplateView{Version: 2}}
	response := httptest.NewRecorder()

	updateTemplateRouter(NewAdminNotificationTemplateHandler(templates)).
		ServeHTTP(response, updateTemplateRequest(updateTemplateBody))

	if response.Code != http.StatusOK || templates.updateCalls != 1 {
		t.Fatalf("status = %d, update calls = %d", response.Code, templates.updateCalls)
	}
}

func TestAdminNotificationTemplateHandlerRejectsAnUnknownKey(t *testing.T) {
	templates := &notificationTemplateServiceStub{updateErr: service.ErrNotificationTemplateNotFound}
	response := httptest.NewRecorder()

	updateTemplateRouter(NewAdminNotificationTemplateHandler(templates)).
		ServeHTTP(response, updateTemplateRequest(updateTemplateBody))

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if code := decodeAPIError(t, response.Body.Bytes()).Code; code != "TEMPLATE_NOT_FOUND" {
		t.Fatalf("code = %q", code)
	}
}

// The SPA highlights the offending input, so every rejected field must arrive
// as a {field, reason} entry under details.fields — the same shape the app
// update policy screen already consumes.
func TestAdminNotificationTemplateHandlerReportsRejectedFields(t *testing.T) {
	templates := &notificationTemplateServiceStub{updateErr: &service.NotificationTemplateValidationError{
		Fields: []service.NotificationTemplateFieldError{
			{Field: "title", Reason: "제목을 입력해 주세요"},
			{Field: "body", Reason: "본문을 입력해 주세요"},
		},
	}}
	response := httptest.NewRecorder()

	updateTemplateRouter(NewAdminNotificationTemplateHandler(templates)).
		ServeHTTP(response, updateTemplateRequest(updateTemplateBody))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Code    string `json:"code"`
		Details struct {
			Fields []service.NotificationTemplateFieldError `json:"fields"`
		} `json:"details"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != "INVALID_TEMPLATE" {
		t.Fatalf("code = %q", payload.Code)
	}
	if len(payload.Details.Fields) != 2 ||
		payload.Details.Fields[0] != (service.NotificationTemplateFieldError{Field: "title", Reason: "제목을 입력해 주세요"}) ||
		payload.Details.Fields[1].Field != "body" {
		t.Fatalf("fields = %#v", payload.Details.Fields)
	}
}

func TestAdminNotificationTemplateHandlerReportsAConcurrentEdit(t *testing.T) {
	templates := &notificationTemplateServiceStub{updateErr: service.ErrNotificationTemplateConflict}
	response := httptest.NewRecorder()

	updateTemplateRouter(NewAdminNotificationTemplateHandler(templates)).
		ServeHTTP(response, updateTemplateRequest(updateTemplateBody))

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	apiError := decodeAPIError(t, response.Body.Bytes())
	if apiError.Code != "TEMPLATE_CONFLICT" || !strings.Contains(apiError.Message, "새로고침") {
		t.Fatalf("error = %#v", apiError)
	}
}

func TestAdminNotificationTemplateHandlerRejectsMalformedBodies(t *testing.T) {
	bodies := map[string]string{
		"unknown field":  `{"title":"제목","body":"본문","expectedVersion":3,"updatedBy":9}`,
		"not an object":  `[]`,
		"trailing value": `{"title":"제목","body":"본문","expectedVersion":3}{}`,
		"wrong type":     `{"title":"제목","body":"본문","expectedVersion":"3"}`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			templates := &notificationTemplateServiceStub{}
			response := httptest.NewRecorder()

			updateTemplateRouter(NewAdminNotificationTemplateHandler(templates)).
				ServeHTTP(response, updateTemplateRequest(body))

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if code := decodeAPIError(t, response.Body.Bytes()).Code; code != "INVALID_BODY" {
				t.Fatalf("code = %q", code)
			}
			if templates.updateCalls != 0 {
				t.Fatalf("update calls = %d, want the write refused", templates.updateCalls)
			}
		})
	}
}

// A missing expectedVersion is a field error, not a malformed body: the service
// owns the concurrency rule, so the omission must reach it and come back marked
// on the form rather than as an opaque INVALID_BODY.
func TestAdminNotificationTemplateHandlerPassesAMissingVersionToTheService(t *testing.T) {
	templates := &notificationTemplateServiceStub{updateErr: &service.NotificationTemplateValidationError{
		Fields: []service.NotificationTemplateFieldError{
			{Field: "expectedVersion", Reason: "편집 중이던 버전 정보가 없습니다"},
		},
	}}
	response := httptest.NewRecorder()

	updateTemplateRouter(NewAdminNotificationTemplateHandler(templates)).
		ServeHTTP(response, updateTemplateRequest(`{"title":"제목","body":"본문"}`))

	if templates.updateCalls != 1 || templates.lastInput.ExpectedVersion != 0 {
		t.Fatalf("update calls = %d, expectedVersion = %d", templates.updateCalls, templates.lastInput.ExpectedVersion)
	}
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if code := decodeAPIError(t, response.Body.Bytes()).Code; code != "INVALID_TEMPLATE" {
		t.Fatalf("code = %q, want the field-level code", code)
	}
}

// An SMS template has no title, so an empty title must reach the service and be
// judged there rather than rejected as a malformed body.
func TestAdminNotificationTemplateHandlerAcceptsAnEmptyTitle(t *testing.T) {
	templates := &notificationTemplateServiceStub{saved: &service.TemplateView{Version: 2}}
	response := httptest.NewRecorder()

	updateTemplateRouter(NewAdminNotificationTemplateHandler(templates)).
		ServeHTTP(response, updateTemplateRequest(`{"title":"","body":"인증번호 {code}","expectedVersion":1}`))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if templates.lastInput.Title != "" || templates.lastInput.Body != "인증번호 {code}" {
		t.Fatalf("input = %#v", templates.lastInput)
	}
}

func TestAdminNotificationTemplateHandlerRequiresAnAuthenticatedOperator(t *testing.T) {
	templates := &notificationTemplateServiceStub{}
	response := httptest.NewRecorder()

	anonymousUpdateTemplateRouter(NewAdminNotificationTemplateHandler(templates)).
		ServeHTTP(response, updateTemplateRequest(updateTemplateBody))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if code := decodeAPIError(t, response.Body.Bytes()).Code; code != "UNAUTHORIZED" {
		t.Fatalf("code = %q", code)
	}
	if templates.updateCalls != 0 {
		t.Fatalf("update calls = %d, want none without an operator", templates.updateCalls)
	}
}

func TestAdminNotificationTemplateHandlerReportsAListFailure(t *testing.T) {
	handler := NewAdminNotificationTemplateHandler(
		&notificationTemplateServiceStub{listErr: errors.New("connection refused")},
	)
	response := httptest.NewRecorder()

	handler.List(response, httptest.NewRequest(http.MethodGet, "/api/admin/notification-templates", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if code := decodeAPIError(t, response.Body.Bytes()).Code; code != "TEMPLATE_FAILED" {
		t.Fatalf("code = %q", code)
	}
}
