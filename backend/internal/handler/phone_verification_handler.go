// phone_verification_handler.go — HTTP handlers for the signup SMS verification flow.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

// PhoneVerificationHandler exposes code request and confirmation for signup.
type PhoneVerificationHandler struct {
	service *service.PhoneVerificationService
	logger  zerolog.Logger
}

// NewPhoneVerificationHandler creates a PhoneVerificationHandler.
func NewPhoneVerificationHandler(svc *service.PhoneVerificationService, logger zerolog.Logger) *PhoneVerificationHandler {
	return &PhoneVerificationHandler{service: svc, logger: logger}
}

// RequestCode sends a one-time code by SMS to the submitted phone number.
func (h *PhoneVerificationHandler) RequestCode(w http.ResponseWriter, r *http.Request) {
	var req model.PhoneVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	result, err := h.service.RequestCode(strings.TrimSpace(req.Phone))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidPhone):
			respondError(w, http.StatusBadRequest, "INVALID_PHONE", "유효한 전화번호를 입력해주세요")
		case errors.Is(err, service.ErrPhoneVerificationThrottled):
			respondError(w, http.StatusTooManyRequests, "PHONE_VERIFICATION_THROTTLED", "인증번호 요청이 너무 많습니다. 잠시 후 다시 시도해주세요")
		default:
			h.logger.Error().Err(err).Msg("phone verification: code request failed")
			respondError(w, http.StatusInternalServerError, "PHONE_VERIFICATION_FAILED", "인증번호 발송에 실패했습니다")
		}
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// ConfirmCode validates a submitted code and returns a short-lived verification token.
func (h *PhoneVerificationHandler) ConfirmCode(w http.ResponseWriter, r *http.Request) {
	var req model.PhoneVerificationConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	req.VerificationID = strings.TrimSpace(req.VerificationID)
	req.Code = strings.TrimSpace(req.Code)
	if req.VerificationID == "" || req.Code == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "인증번호를 입력해주세요")
		return
	}

	result, err := h.service.ConfirmCode(req.VerificationID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPhoneVerificationNotFound):
			respondError(w, http.StatusBadRequest, "PHONE_VERIFICATION_EXPIRED", "인증 요청이 만료되었습니다. 인증번호를 다시 요청해주세요")
		case errors.Is(err, service.ErrPhoneVerificationCodeMismatch):
			respondError(w, http.StatusBadRequest, "PHONE_VERIFICATION_MISMATCH", "인증번호가 일치하지 않습니다")
		case errors.Is(err, service.ErrPhoneVerificationAttemptsExceeded):
			respondError(w, http.StatusTooManyRequests, "PHONE_VERIFICATION_ATTEMPTS", "입력 횟수를 초과했습니다. 인증번호를 다시 요청해주세요")
		default:
			h.logger.Error().Err(err).Msg("phone verification: confirm failed")
			respondError(w, http.StatusInternalServerError, "PHONE_VERIFICATION_FAILED", "인증 확인에 실패했습니다")
		}
		return
	}
	respondJSON(w, http.StatusOK, result)
}
