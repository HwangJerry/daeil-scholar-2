// auth_social_handler.go — Shared social OAuth callback orchestration
package handler

import (
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

// handleSocialCallback is the shared callback logic after a social OAuth token exchange.
// Decision order:
//  1. Existing social link (WEO_MEMBER_SOCIAL) → login as that member.
//  2. No match → persist a one-time canonical continuation and require account reauthentication.
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

	linkRequired, err := h.service.BeginSocialLinkContinuation(
		model.SocialProvider(gate),
		info.KakaoID,
		info.Email,
		info.Nickname,
		info.ProfileImageURL,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LINK_FAILED", "Failed to start social account link")
		return
	}
	http.Redirect(w, r, h.cfg.Server.AllowedOrigin+"/login/link?token="+linkRequired.LinkToken, http.StatusFound)
}

// completeSocialLogin handles the final login step for an existing social link.
func (h *AuthHandler) completeSocialLogin(w http.ResponseWriter, r *http.Request, gate string, user *model.User, accessToken string) {
	if err := h.service.LoginWithBridge(user, w, r); err != nil {
		if errors.Is(err, service.ErrLoginSuspended) || errors.Is(err, service.ErrLoginWithdrawn) {
			respondError(w, http.StatusForbidden, service.LoginErrorCode(err), "이 계정은 현재 로그인할 수 없습니다.")
			return
		}
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Failed to login")
		return
	}
	if gate == "KT" {
		h.service.CacheKakaoToken(user.USRSeq, accessToken)
	}
	http.Redirect(w, r, h.cfg.Server.AllowedOrigin+"/", http.StatusFound)
}
