// admin_notification_template_handler.go — Administrator endpoints for editable notification texts.
package handler

import (
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

// NotificationTemplateServicer is the slice of the template service the admin
// screen needs: the editable list and the write, which returns the saved row.
type NotificationTemplateServicer interface {
	List() ([]service.TemplateView, error)
	Update(input service.UpdateNotificationTemplateInput) (*service.TemplateView, error)
}

type AdminNotificationTemplateHandler struct {
	templates NotificationTemplateServicer
}

func NewAdminNotificationTemplateHandler(templates NotificationTemplateServicer) *AdminNotificationTemplateHandler {
	return &AdminNotificationTemplateHandler{templates: templates}
}

func (h *AdminNotificationTemplateHandler) List(w http.ResponseWriter, _ *http.Request) {
	views, err := h.templates.List()
	if err != nil {
		respondNotificationTemplateError(w, err)
		return
	}
	if views == nil {
		views = []service.TemplateView{}
	}
	respondJSON(w, http.StatusOK, views)
}

type updateNotificationTemplateRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	// ExpectedVersion is the version the administrator was editing. The service
	// rejects a missing one as a field error, so a lost concurrency check is
	// reported on the form rather than as a malformed body.
	ExpectedVersion int `json:"expectedVersion"`
}

func (h *AdminNotificationTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	var request updateNotificationTemplateRequest
	if err := decodeClosedJSON(r, &request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "title, body, expectedVersion가 필요합니다")
		return
	}
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	// The saved row comes back from the write itself, so the screen picks up the
	// new version without a second request that could fail after a committed
	// save or show a rival administrator's edit.
	view, err := h.templates.Update(service.UpdateNotificationTemplateInput{
		Key:             chi.URLParam(r, "key"),
		Title:           request.Title,
		Body:            request.Body,
		ExpectedVersion: request.ExpectedVersion,
		UpdatedBy:       user.USRSeq,
	})
	if err != nil {
		respondNotificationTemplateError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, view)
}

// respondNotificationTemplateError maps template errors onto the codes the
// admin SPA reacts to: field-level 400, unknown key 404, concurrent edit 409.
func respondNotificationTemplateError(w http.ResponseWriter, err error) {
	var validation *service.NotificationTemplateValidationError
	switch {
	case errors.As(err, &validation):
		respondJSON(w, http.StatusBadRequest, model.APIError{
			Code:    "INVALID_TEMPLATE",
			Message: "알림 문구를 저장할 수 없습니다",
			Details: map[string][]service.NotificationTemplateFieldError{"fields": validation.Fields},
		})
	case errors.Is(err, service.ErrNotificationTemplateNotFound):
		respondError(w, http.StatusNotFound, "TEMPLATE_NOT_FOUND", "알림 문구를 찾을 수 없습니다")
	case errors.Is(err, service.ErrNotificationTemplateConflict):
		respondError(w, http.StatusConflict, "TEMPLATE_CONFLICT",
			"다른 관리자가 먼저 수정했습니다. 새로고침 후 다시 시도해 주세요")
	default:
		respondError(w, http.StatusInternalServerError, "TEMPLATE_FAILED", "알림 문구 처리에 실패했습니다")
	}
}
