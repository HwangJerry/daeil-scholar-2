package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
)

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
