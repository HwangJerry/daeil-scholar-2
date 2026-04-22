// auth_social_handler.go — Shared social OAuth callback orchestration
package handler

import (
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

// handleSocialCallback is the shared callback logic after a social OAuth token exchange.
// Decision order:
//  1. Existing social link (WEO_MEMBER_SOCIAL) → login as that member.
//  2. Email match on WEO_MEMBER → insert social link + login (no form).
//  3. No match → cache provider data and redirect to /login/link for the signup form.
func (h *AuthHandler) handleSocialCallback(w http.ResponseWriter, r *http.Request, gate string, info service.KakaoUserInfo) {
	user, err := h.service.FindMemberBySocialID(gate, info.KakaoID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LOOKUP_FAILED", "Failed to lookup member")
		return
	}
	if user != nil {
		h.completeSocialLogin(w, r, gate, user, info.AccessToken)
		return
	}

	if info.Email != "" {
		matched, err := h.service.FindMemberByEmail(info.Email)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "LOOKUP_FAILED", "Failed to lookup member by email")
			return
		}
		if matched != nil {
			if err := h.service.InsertSocialLink(matched.USRSeq, gate, info.KakaoID, info.Email); err != nil {
				h.logger.Error().Err(err).Int("usrSeq", matched.USRSeq).Msg("kakao: insert social link failed")
				respondError(w, http.StatusInternalServerError, "LINK_FAILED", "Failed to link social account")
				return
			}
			if info.ProfileImageURL != "" {
				if err := h.service.UpdateProfilePhotoIfEmpty(matched.USRSeq, info.ProfileImageURL); err != nil {
					h.logger.Warn().Err(err).Int("usrSeq", matched.USRSeq).Msg("kakao: optional photo update failed")
				}
			}
			h.completeSocialLogin(w, r, gate, matched, info.AccessToken)
			return
		}
	}

	linkToken := h.service.GenerateSessionID()
	h.cache.Set("social_link:"+linkToken, model.SocialLinkData{
		Provider:        gate,
		SocialID:        info.KakaoID,
		Email:           info.Email,
		Nickname:        info.Nickname,
		ProfileImageURL: info.ProfileImageURL,
		AccessToken:     info.AccessToken,
	}, 5*time.Minute)
	http.Redirect(w, r, h.cfg.Server.AllowedOrigin+"/login/link?token="+linkToken, http.StatusFound)
}

// completeSocialLogin handles the final login step shared by "existing social link" and "email-matched" paths.
// BBB (pending approval) accounts are redirected to the login page with an error flag.
func (h *AuthHandler) completeSocialLogin(w http.ResponseWriter, r *http.Request, gate string, user *model.User, accessToken string) {
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
}
