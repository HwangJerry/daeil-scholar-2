package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

const deferredLogoutBodyLimit = 16 * 1024

// DeferredLogout accepts only the original refresh proof, never caller-selected
// account/SID fields. No session cookies or credentials are issued by this route.
func (h *AuthHandler) DeferredLogout(w http.ResponseWriter, r *http.Request) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		respondError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "JSON 요청이 필요합니다.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, deferredLogoutBodyLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request model.RefreshTokenRequest
	err = decoder.Decode(&request)
	if err == nil {
		var trailing any
		err = decoder.Decode(&trailing)
		if err == io.EOF {
			err = nil
		} else if err == nil {
			err = errors.New("trailing JSON")
		}
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "요청이 너무 큽니다.")
		} else {
			respondError(w, http.StatusBadRequest, "INVALID_BODY", "올바른 JSON 요청이 필요합니다.")
		}
		return
	}
	if request.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "refresh token is required")
		return
	}
	if err := h.service.RevokeEndedMobileSession(r.Context(), request.RefreshToken); err != nil {
		if errors.Is(err, repository.ErrRefreshTokenInvalid) {
			respondError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "유효하지 않은 종료 증명입니다.")
		} else {
			// Database errors can contain parameters: log only this bounded event code.
			h.logger.Error().Str("code", "LOGOUT_FAILED").Msg("deferred session revocation failed")
			respondError(w, http.StatusInternalServerError, "LOGOUT_FAILED", "세션 종료를 다시 시도해주세요.")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
