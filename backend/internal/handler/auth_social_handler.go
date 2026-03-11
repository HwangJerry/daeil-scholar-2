// auth_social_handler.go — Shared social OAuth callback orchestration
package handler

import (
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

// handleSocialCallback is the shared callback logic: find existing member or redirect to link form.
func (h *AuthHandler) handleSocialCallback(w http.ResponseWriter, r *http.Request, gate, socialID, email, nickname, accessToken string) {
	user, err := h.service.FindMemberBySocialID(gate, socialID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LOOKUP_FAILED", "Failed to lookup member")
		return
	}
	if user != nil {
		if user.USRStatus == "BBB" {
			http.Redirect(w, r, h.cfg.Server.AllowedOrigin+"/login?error=pending_approval", http.StatusFound)
			return
		}
		if err := h.service.LoginWithBridge(user, w, r); err != nil {
			respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Failed to login")
			return
		}
		if gate == "KT" {
			h.service.CacheKakaoToken(user.USRSeq, accessToken)
		}
		http.Redirect(w, r, h.cfg.Server.AllowedOrigin+"/", http.StatusFound)
		return
	}

	linkToken := h.service.GenerateSessionID()
	h.cache.Set("social_link:"+linkToken, model.SocialLinkData{
		Provider:    gate,
		SocialID:    socialID,
		Email:       email,
		Nickname:    nickname,
		AccessToken: accessToken,
	}, 5*time.Minute)
	http.Redirect(w, r, h.cfg.Server.AllowedOrigin+"/login/link?token="+linkToken, http.StatusFound)
}
