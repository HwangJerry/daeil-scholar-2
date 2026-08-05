// auth_kakao_mobile_handler.go — Mobile-first Kakao login endpoint.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

func (h *AuthHandler) KakaoMobileLogin(w http.ResponseWriter, r *http.Request) {
	var req model.KakaoMobileLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	grantType := strings.ToLower(strings.TrimSpace(req.GrantType))
	var result model.SocialAuthResult
	var err error
	switch grantType {
	case "access_token":
		if strings.TrimSpace(req.AccessToken) == "" {
			respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "accessToken is required")
			return
		}
		result, err = h.service.AuthenticateKakaoMobile(r.Context(), req.AccessToken)
	default:
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "grantType must be access_token")
		return
	}

	if err != nil {
		if errors.Is(err, service.ErrKakaoVerificationFailed) {
			h.logger.Warn().Msg("kakao mobile: provider verification failed")
			respondError(w, http.StatusUnauthorized, "KAKAO_VERIFICATION_FAILED", "Kakao login failed")
			return
		}
		if errors.Is(err, service.ErrLoginSuspended) || errors.Is(err, service.ErrLoginWithdrawn) {
			respondError(w, http.StatusForbidden, service.LoginErrorCode(err), "이 계정은 현재 로그인할 수 없습니다.")
			return
		}
		h.logger.Error().Msg("kakao mobile: authentication failed")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Kakao login failed")
		return
	}
	writeMobileAuthResult(w, result)
}
