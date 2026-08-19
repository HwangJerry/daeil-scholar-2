// auth_kakao_mobile_handler.go — Mobile-first Kakao login endpoint.
package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

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
	if grantType == "" {
		grantType = "access_token"
	}

	var info service.KakaoUserInfo
	var err error
	switch grantType {
	case "access_token", "token", "bearer":
		if strings.TrimSpace(req.AccessToken) == "" {
			respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "accessToken is required")
			return
		}
		info, err = h.service.GetKakaoProfileByAccessToken(req.AccessToken)
	case "authorization_code", "code":
		if strings.TrimSpace(req.Code) == "" {
			respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "code is required")
			return
		}
		redirectURI := strings.TrimSpace(req.RedirectURI)
		if redirectURI == "" {
			redirectURI = h.cfg.Kakao.RedirectURI
		}
		info, err = h.service.ExchangeKakaoTokenWithRedirect(req.Code, redirectURI)
	default:
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "grantType must be one of access_token, authorization_code")
		return
	}

	if err != nil {
		h.logger.Error().Err(err).Msg("kakao mobile: token exchange/lookup failed")
		respondError(w, http.StatusBadRequest, "KAKAO_EXCHANGE_FAILED", "Kakao login failed")
		return
	}

	if user, err := h.service.FindMemberBySocialID("KT", info.KakaoID); err != nil {
		h.logger.Error().Err(err).Int("kakaoId", len(info.KakaoID)).Msg("kakao mobile: lookup failed")
		respondError(w, http.StatusInternalServerError, "LOOKUP_FAILED", "Failed to lookup member")
		return
	} else if user != nil {
		h.completeKakaoMobileLogin(w, r, user, info.AccessToken)
		return
	}

	// Per CLAUDE.md/docs/spec-auth.md §4.6: mobile deliberately does NOT
	// implement account-linking. A Kakao identity with no existing social-ID
	// link always gets 202 link_required here - even when its email matches
	// an existing member - so the app can surface "sign in with your
	// existing ID" rather than silently auto-linking accounts by email alone
	// (email match is not proof of ownership of the alumni account).
	linkToken := h.service.GenerateSessionID()
	if linkToken == "" {
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Failed to start social link")
		return
	}
	h.cache.Set("social_link:"+linkToken, model.SocialLinkData{
		Provider:        "KT",
		SocialID:        info.KakaoID,
		Email:           info.Email,
		Nickname:        info.Nickname,
		ProfileImageURL: info.ProfileImageURL,
		AccessToken:     info.AccessToken,
	}, 5*time.Minute)
	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":          "link_required",
		"error":           "SOCIAL_LINK_REQUIRED",
		"provider":        "KT",
		"linkToken":       linkToken,
		"socialId":        info.KakaoID,
		"email":           info.Email,
		"nickname":        info.Nickname,
		"profileImageUrl": info.ProfileImageURL,
	})
}

func (h *AuthHandler) completeKakaoMobileLogin(w http.ResponseWriter, r *http.Request, user *model.User, kakaoAccessToken string) {
	// Mirrors auth_social_handler.go's completeSocialLogin: a Kakao identity
	// already being linked to a member does not mean that member is currently
	// allowed to log in (withdrawn/suspended/pending-approval accounts must
	// still be rejected here, same as the password and web social paths).
	if err := (service.LoginEligibilityPolicy{}).EnsureLoginAllowed(user); err != nil {
		respondError(w, http.StatusForbidden, service.LoginErrorCode(err), "이 계정은 현재 로그인할 수 없습니다.")
		return
	}

	authUser := model.AuthUser{USRSeq: user.USRSeq, USRID: user.USRID, USRName: user.USRName, USRStatus: user.USRStatus}

	// Store the Kakao credential before issuing any session state, matching
	// the web login path (auth_social_handler.go): if vault storage fails we
	// must fail closed without having already persisted an active refresh
	// session that the user has no way to know about or revoke.
	if kakaoAccessToken != "" {
		if err := h.socialLifecycle.StoreCredential(user.USRSeq, model.SocialProviderKakao, kakaoAccessToken); err != nil {
			h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Str("provider", string(model.SocialProviderKakao)).Msg("kakao mobile: social credential storage failed")
			respondError(w, http.StatusServiceUnavailable, "CREDENTIAL_STORAGE_UNAVAILABLE", "로그인을 완료할 수 없습니다.")
			return
		}
	}

	mobileSessionID := h.service.GenerateSessionID()
	if mobileSessionID == "" {
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 토큰 생성에 실패했습니다")
		return
	}
	mobileToken, err := h.service.GenerateMobileJWT(&authUser, mobileSessionID)
	if err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("kakao mobile: access token issue failed")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 토큰 재발급에 실패했습니다")
		return
	}
	refreshToken, refreshJTI, refreshExpiresAt, err := h.service.GenerateMobileRefreshJWT(&authUser, mobileSessionID)
	if err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("kakao mobile: refresh token issue failed")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 토큰 재발급에 실패했습니다")
		return
	}
	if err := h.service.RecordMobileRefreshToken(authUser.USRSeq, mobileSessionID, refreshJTI, refreshExpiresAt); err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("kakao mobile: failed to persist refresh token")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 토큰 재발급에 실패했습니다")
		return
	}
	if kakaoAccessToken != "" {
		h.service.CacheKakaoToken(user.USRSeq, kakaoAccessToken)
	}

	now := time.Now()
	respondJSON(w, http.StatusOK, struct {
		USRSeq           int    `json:"usrSeq"`
		USRID            string `json:"usrId"`
		USRName          string `json:"usrName"`
		USRStatus        string `json:"usrStatus"`
		AccessToken      string `json:"accessToken"`
		RefreshToken     string `json:"refreshToken"`
		AccessIssuedAt   int64  `json:"accessIssuedAt"`
		AccessExpiresAt  int64  `json:"accessExpiresAt"`
		RefreshExpiresAt int64  `json:"refreshExpiresAt"`
		Sid              string `json:"sid"`
		Jti              string `json:"jti"`
	}{
		USRSeq:           authUser.USRSeq,
		USRID:            authUser.USRID,
		USRName:          authUser.USRName,
		USRStatus:        authUser.USRStatus,
		AccessToken:      mobileToken,
		RefreshToken:     refreshToken,
		AccessIssuedAt:   now.Unix(),
		AccessExpiresAt:  now.Add(h.service.MobileAccessTokenTTL()).Unix(),
		RefreshExpiresAt: refreshExpiresAt.Unix(),
		Sid:              mobileSessionID,
		Jti:              refreshJTI,
	})
}
