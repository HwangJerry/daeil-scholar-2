package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
)

func (h *AuthHandler) MobileLogin(w http.ResponseWriter, r *http.Request) {
	var request model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	email := strings.TrimSpace(request.Email)
	usrID := strings.TrimSpace(request.USRID)
	if (email == "" && usrID == "") || request.Password == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "이메일과 비밀번호를 입력하세요")
		return
	}
	var user *model.User
	var err error
	if email != "" {
		user, err = h.memberSvc.LoginWithEmailPassword(email, request.Password)
	} else {
		user, err = h.memberSvc.LoginWithPassword(usrID, request.Password)
	}
	if err != nil {
		if isLoginPolicyError(err) {
			respondError(w, http.StatusForbidden, service.LoginErrorCode(err), "이 계정은 현재 로그인할 수 없습니다.")
			return
		}
		h.logger.Error().Err(err).Msg("mobile password login failed")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 처리 중 오류가 발생했습니다")
		return
	}
	if user == nil {
		respondError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "이메일 또는 비밀번호가 올바르지 않습니다")
		return
	}
	session, err := h.mobileIssuer.Issue(user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 토큰 발급에 실패했습니다")
		return
	}
	respondJSON(w, http.StatusOK, model.SocialAuthResult{
		Status:  model.SocialAuthAuthenticated,
		Session: session,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var request model.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	refreshToken := strings.TrimSpace(request.RefreshToken)
	if refreshToken == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "refresh token is required")
		return
	}
	session, err := h.mobileIssuer.Rotate(refreshToken)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRefreshTokenReplay):
			respondError(w, http.StatusUnauthorized, "REFRESH_REPLAY_DETECTED", "세션을 다시 시작해주세요.")
		case errors.Is(err, repository.ErrRefreshTokenInvalid):
			respondError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "세션을 다시 시작해주세요.")
		case isLoginPolicyError(err):
			respondError(w, http.StatusForbidden, service.LoginErrorCode(err), "이 계정은 현재 로그인할 수 없습니다.")
		default:
			h.logger.Error().Err(err).Msg("refresh rotation failed")
			respondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "토큰 갱신에 실패했습니다.")
		}
		return
	}
	respondJSON(w, http.StatusOK, session)
}

func (h *AuthHandler) KakaoMobileLogin(w http.ResponseWriter, r *http.Request) {
	var request model.KakaoMobileLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	accessToken, err := h.resolveKakaoMobileAccessToken(request)
	if err != nil {
		respondError(w, http.StatusBadRequest, "KAKAO_EXCHANGE_FAILED", "Kakao login failed")
		return
	}
	result, err := h.socialAuth.Authenticate(
		r.Context(),
		model.KakaoAuthorization{
			AccessToken: accessToken,
		},
	)
	if err != nil {
		h.logger.Warn().Err(err).Msg("kakao mobile verification failed")
		respondError(w, http.StatusUnauthorized, "KAKAO_VERIFICATION_FAILED", "Kakao login failed")
		return
	}
	writeMobileAuthResult(w, result)
}

func (h *AuthHandler) resolveKakaoMobileAccessToken(request model.KakaoMobileLoginRequest) (string, error) {
	grantType := strings.ToLower(strings.TrimSpace(request.GrantType))
	if grantType == "" || grantType == "access_token" || grantType == "token" || grantType == "bearer" {
		if strings.TrimSpace(request.AccessToken) == "" {
			return "", errors.New("access token is required")
		}
		return request.AccessToken, nil
	}
	if grantType != "authorization_code" && grantType != "code" {
		return "", errors.New("unsupported grant type")
	}
	redirectURI := strings.TrimSpace(request.RedirectURI)
	if redirectURI == "" {
		redirectURI = h.cfg.Kakao.RedirectURI
	}
	if !containsString(h.cfg.Kakao.AllowedRedirectURIs, redirectURI) {
		return "", errors.New("redirect URI is not allowed")
	}
	info, err := h.service.ExchangeKakaoTokenWithRedirect(request.Code, redirectURI)
	if err != nil {
		return "", err
	}
	return info.AccessToken, nil
}

func writeMobileAuthResult(w http.ResponseWriter, result model.SocialAuthResult) {
	switch result.Status {
	case model.SocialAuthAuthenticated:
		if result.Session == nil {
			respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Missing session")
			return
		}
		respondJSON(w, http.StatusOK, result)
	case model.SocialAuthLinkRequired:
		if result.LinkRequired == nil {
			respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Missing link context")
			return
		}
		respondJSON(w, http.StatusAccepted, result)
	case model.SocialAuthRejected:
		if result.Rejected == nil {
			respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Missing rejection context")
			return
		}
		respondError(w, http.StatusForbidden, result.Rejected.Code, result.Rejected.Message)
	default:
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Invalid authentication result")
	}
}

func isLoginPolicyError(err error) bool {
	return errors.Is(err, service.ErrLoginPending) ||
		errors.Is(err, service.ErrLoginSuspended) ||
		errors.Is(err, service.ErrLoginWithdrawn)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
