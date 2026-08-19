// auth_social_handler.go — Shared social OAuth callback orchestration
package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

// handleSocialCallback is the shared callback logic after a social OAuth token exchange.
// Only an existing (provider, subject) link can log in immediately. Provider
// email is display/prefill data and is never an account-linking credential.
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

	linkToken := h.service.GenerateSessionID()
	if _, err := h.socialLinkTokens.Put(linkToken, model.SocialLinkData{
		Provider:        gate,
		SocialID:        info.KakaoID,
		Email:           info.Email,
		Nickname:        info.Nickname,
		ProfileImageURL: info.ProfileImageURL,
		AccessToken:     info.AccessToken,
	}, service.SocialLinkTokenTTL); err != nil {
		respondError(w, http.StatusInternalServerError, "LINK_TOKEN_FAILED", "Failed to start account linking")
		return
	}
	http.Redirect(w, r, h.cfg.Server.AllowedOrigin+"/login/link?token="+linkToken, http.StatusFound)
}

// completeSocialLogin applies the shared eligibility policy before issuing a web session.
func (h *AuthHandler) completeSocialLogin(w http.ResponseWriter, r *http.Request, gate string, user *model.User, accessToken string) {
	if err := (service.LoginEligibilityPolicy{}).EnsureLoginAllowed(user); err != nil {
		http.Redirect(w, r, h.cfg.Server.AllowedOrigin+"/login?error="+service.LoginErrorCode(err), http.StatusFound)
		return
	}
	if accessToken != "" {
		if err := h.socialLifecycle.StoreCredential(user.USRSeq, model.SocialProvider(gate), accessToken); err != nil {
			h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Str("provider", gate).Msg("social credential storage failed")
			respondError(w, http.StatusServiceUnavailable, "CREDENTIAL_STORAGE_UNAVAILABLE", "소셜 로그인을 완료할 수 없습니다.")
			return
		}
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
