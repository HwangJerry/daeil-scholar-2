// password_reset_handler.go — HTTP handlers for password reset API endpoints
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

// PasswordResetHandler handles password reset HTTP requests.
type PasswordResetHandler struct {
	service *service.PasswordResetService
	logger  zerolog.Logger
}

// NewPasswordResetHandler creates a new PasswordResetHandler.
func NewPasswordResetHandler(svc *service.PasswordResetService, logger zerolog.Logger) *PasswordResetHandler {
	return &PasswordResetHandler{service: svc, logger: logger}
}

// RequestReset handles POST /api/auth/password/reset-request.
// Accepts an email and initiates the password reset flow.
func (h *PasswordResetHandler) RequestReset(w http.ResponseWriter, r *http.Request) {
	var req model.PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if err := h.service.RequestReset(req); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ConfirmReset handles POST /api/auth/password/reset-confirm.
// Validates the token and sets the new password.
func (h *PasswordResetHandler) ConfirmReset(w http.ResponseWriter, r *http.Request) {
	var req model.PasswordResetConfirm
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if err := h.service.ConfirmReset(req); err != nil {
		respondError(w, http.StatusBadRequest, "RESET_FAILED", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ValidateToken handles GET /api/auth/password/validate-token?token=...
// Returns whether the token is valid and the associated member name.
func (h *PasswordResetHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	result, err := h.service.ValidateToken(token)
	if err != nil {
		h.logger.Error().Err(err).Msg("validate token failed")
		respondError(w, http.StatusInternalServerError, "VALIDATE_FAILED", "토큰 검증에 실패했습니다")
		return
	}

	respondJSON(w, http.StatusOK, result)
}
