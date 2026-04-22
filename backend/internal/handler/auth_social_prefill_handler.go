// auth_social_prefill_handler.go — Read-only access to cached social-link data for signup form prefill
package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
)

type socialLinkPrefillResponse struct {
	Provider        string `json:"provider"`
	Email           string `json:"email"`
	Nickname        string `json:"nickname"`
	ProfileImageURL string `json:"profileImageUrl"`
}

// SocialLinkPrefill returns the cached provider info (email, nickname, profile image)
// so the signup form can prefill user-facing fields. The cache is NOT consumed here
// so the caller can still POST /api/auth/social/link afterward.
func (h *AuthHandler) SocialLinkPrefill(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "MISSING_TOKEN", "token 파라미터가 필요합니다")
		return
	}
	cached, found := h.cache.Get("social_link:" + token)
	if !found {
		respondError(w, http.StatusNotFound, "TOKEN_NOT_FOUND", "유효한 소셜 링크 토큰이 아닙니다")
		return
	}
	data, ok := cached.(model.SocialLinkData)
	if !ok {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Invalid cached data")
		return
	}
	respondJSON(w, http.StatusOK, socialLinkPrefillResponse{
		Provider:        data.Provider,
		Email:           data.Email,
		Nickname:        data.Nickname,
		ProfileImageURL: data.ProfileImageURL,
	})
}
