package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

func (h *AuthHandler) DisconnectSocialProvider(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	provider := parseSocialProvider(chi.URLParam(r, "provider"))
	if !provider.Valid() {
		respondError(w, http.StatusBadRequest, "INVALID_PROVIDER", "지원하지 않는 소셜 제공자입니다.")
		return
	}
	result, err := h.socialLifecycle.Disconnect(r.Context(), user.USRSeq, provider)
	if errors.Is(err, service.ErrLastLoginMethod) {
		respondError(w, http.StatusConflict, "LAST_LOGIN_METHOD", "마지막 로그인 수단은 연결 해제할 수 없습니다.")
		return
	}
	if err != nil {
		h.logger.Warn().Err(err).Int("usrSeq", user.USRSeq).Str("provider", string(provider)).Msg("social disconnect deferred")
		respondError(w, http.StatusServiceUnavailable, "PROVIDER_REVOCATION_PENDING", "제공자 연결 해제를 재시도할 예정입니다.")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) AccountConnections(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	connections, err := h.socialLifecycle.Connections(user.USRSeq)
	if err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("account connections lookup failed")
		respondError(w, http.StatusInternalServerError, "CONNECTIONS_FAILED", "로그인 연결 정보를 불러올 수 없습니다.")
		return
	}
	respondJSON(w, http.StatusOK, connections)
}

func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	result, err := h.socialLifecycle.DeleteAccount(r.Context(), user.USRSeq)
	if err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("account deactivation failed")
		respondError(w, http.StatusInternalServerError, "ACCOUNT_DELETE_FAILED", "회원 탈퇴를 처리할 수 없습니다.")
		return
	}
	logoutErr := h.service.LogoutAll(w, user.USRSeq)
	if logoutErr != nil {
		result.RevocationPending = true
	}
	if result.RevocationPending {
		h.logger.Warn().Err(logoutErr).Int("usrSeq", user.USRSeq).Msg("account disabled with provider revocation pending")
		respondJSON(w, http.StatusAccepted, map[string]interface{}{
			"status":            "deleted",
			"revocationPending": true,
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":            "deleted",
		"revocationPending": false,
	})
}

func parseSocialProvider(value string) model.SocialProvider {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "kakao", "kt":
		return model.SocialProviderKakao
	case "apple", "ap":
		return model.SocialProviderApple
	default:
		return ""
	}
}
