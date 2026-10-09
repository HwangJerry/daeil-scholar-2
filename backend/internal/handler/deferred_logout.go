package handler

import (
	"encoding/json"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"io"
	"mime"
	"net/http"
)

const deferredLogoutBodyLimit = 16 * 1024

func (h *AuthHandler) DeferredLogout(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RefreshToken string  `json:"refreshToken"`
		DeviceToken  *string `json:"deviceToken,omitempty"`
	}
	if !decodeDeferredLogoutJSON(w, r, &request) {
		return
	}
	if request.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "refresh token is required")
		return
	}
	device := ""
	if request.DeviceToken != nil {
		device = *request.DeviceToken
		if !service.ValidPushDeviceToken(device) {
			respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "기기 토큰이 올바르지 않습니다.")
			return
		}
	}
	confirmed, err := h.service.RevokeEndedMobileSessionWithDeviceResult(r.Context(), request.RefreshToken, device)
	if err == nil {
		result := "unconfirmed"
		if confirmed {
			result = "confirmed"
		}
		w.Header().Set("X-Session-Logout-Result", result)
	}
	h.writeDeferredLogoutOutcome(w, err, false)
}

// DeferredLogoutAll is proof-authorized once; never refresh or retry using revoked proof.
func (h *AuthHandler) DeferredLogoutAll(w http.ResponseWriter, r *http.Request) {
	var request model.RefreshTokenRequest
	if !decodeDeferredLogoutJSON(w, r, &request) {
		return
	}
	if request.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "refresh token is required")
		return
	}
	h.writeDeferredLogoutOutcome(w, h.service.RevokeAllSessionsWithOriginalProof(r.Context(), request.RefreshToken), true)
}

func decodeDeferredLogoutJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		respondError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "JSON 요청이 필요합니다.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, deferredLogoutBodyLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(target)
	if err == nil {
		var trailing any
		err = decoder.Decode(&trailing)
		if err == io.EOF {
			err = nil
		} else if err == nil {
			err = errors.New("trailing JSON")
		}
	}
	if err == nil {
		return true
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		respondError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "요청이 너무 큽니다.")
	} else {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "올바른 JSON 요청이 필요합니다.")
	}
	return false
}

func (h *AuthHandler) writeDeferredLogoutOutcome(w http.ResponseWriter, err error, global bool) {
	if err == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if errors.Is(err, repository.ErrRefreshTokenInvalid) {
		respondError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "유효하지 않은 종료 증명입니다.")
		return
	}
	// Parameter-bearing DB errors and proofs must never enter logs.
	h.logger.Error().Str("code", "LOGOUT_FAILED").Bool("global", global).Msg("deferred session revocation failed")
	message := "세션 종료를 다시 시도해주세요."
	if global {
		message = "모든 기기의 로그아웃 결과를 확인하지 못했습니다."
	}
	respondError(w, http.StatusInternalServerError, "LOGOUT_FAILED", message)
}
