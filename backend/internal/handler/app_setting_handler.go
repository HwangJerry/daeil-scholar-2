// app_setting_handler.go — Public and administrator HTTP endpoints for application settings.
package handler

import (
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AppSettingServicer interface {
	ListSettings() ([]model.AppSetting, error)
	GetPublicSettings() (map[string]string, error)
	UpdateValue(key, value string, updatedBy int) error
}

type AppSettingHandler struct {
	service AppSettingServicer
}

func NewAppSettingHandler(appSettingService AppSettingServicer) *AppSettingHandler {
	return &AppSettingHandler{service: appSettingService}
}

func (h *AppSettingHandler) List(w http.ResponseWriter, _ *http.Request) {
	settings, err := h.service.ListSettings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "SETTINGS_FAILED", "Failed to load app settings")
		return
	}
	respondJSON(w, http.StatusOK, settings)
}

func (h *AppSettingHandler) Public(w http.ResponseWriter, _ *http.Request) {
	settings, err := h.service.GetPublicSettings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "SETTINGS_FAILED", "Failed to load public app settings")
		return
	}
	respondJSON(w, http.StatusOK, settings)
}

type updateAppSettingRequest struct {
	Value *string `json:"value"`
}

func (h *AppSettingHandler) Update(w http.ResponseWriter, r *http.Request) {
	var request updateAppSettingRequest
	if err := decodeClosedJSON(r, &request); err != nil || request.Value == nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "value must be a string")
		return
	}
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	err := h.service.UpdateValue(chi.URLParam(r, "key"), *request.Value, user.USRSeq)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, service.ErrInvalidAppSettingKey), errors.Is(err, service.ErrInvalidAppSettingValue):
		respondError(w, http.StatusBadRequest, "INVALID_SETTING", err.Error())
	case errors.Is(err, service.ErrAppSettingNotFound):
		respondError(w, http.StatusNotFound, "SETTING_NOT_FOUND", "App setting not found")
	default:
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update app setting")
	}
}
