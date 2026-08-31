package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

func (h *AuthHandler) GetAccountConnections(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	connections, err := h.service.GetAccountConnections(user.USRSeq)
	if err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("account connections lookup failed")
		respondError(w, http.StatusInternalServerError, "CONNECTIONS_LOOKUP_FAILED", "로그인 수단을 조회할 수 없습니다.")
		return
	}
	respondJSON(w, http.StatusOK, connections)
}

func (h *AuthHandler) LinkIdentity(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	provider := chi.URLParam(r, "provider")
	authorization, err := socialLinkAuthorization(provider, r)
	if err != nil {
		if errors.Is(err, service.ErrInvalidSocialProvider) {
			respondError(w, http.StatusBadRequest, "INVALID_PROVIDER", "지원하지 않는 소셜 로그인 수단입니다.")
			return
		}
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	connections, err := h.socialAuth.LinkIdentity(r.Context(), user.USRSeq, provider, authorization)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidSocialProvider):
			respondError(w, http.StatusBadRequest, "INVALID_PROVIDER", "지원하지 않는 소셜 로그인 수단입니다.")
		case errors.Is(err, service.ErrSocialAccountAlreadyLinked):
			respondError(w, http.StatusConflict, "SOCIAL_ACCOUNT_ALREADY_LINKED", "이미 다른 계정에 연결된 소셜 로그인입니다.")
		case errors.Is(err, service.ErrSocialIdentityVerification):
			respondError(w, http.StatusUnauthorized, "SOCIAL_VERIFICATION_FAILED", "소셜 계정을 확인할 수 없습니다.")
		case errors.Is(err, service.ErrSocialCredentialStorage):
			respondError(w, http.StatusServiceUnavailable, "CREDENTIAL_STORAGE_UNAVAILABLE", "소셜 계정 연결을 완료할 수 없습니다.")
		default:
			h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Str("provider", provider).Msg("social identity link failed")
			respondError(w, http.StatusInternalServerError, "IDENTITY_LINK_FAILED", "소셜 로그인 수단을 연결할 수 없습니다.")
		}
		return
	}
	respondJSON(w, http.StatusOK, connections)
}

func socialLinkAuthorization(provider string, r *http.Request) (model.SocialAuthorization, error) {
	switch provider {
	case "kakao":
		var request model.KakaoMobileLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return nil, errors.New("잘못된 요청 본문입니다.")
		}
		if strings.TrimSpace(request.AccessToken) == "" {
			return nil, errors.New("accessToken이 필요합니다.")
		}
		return model.KakaoAuthorization{AccessToken: request.AccessToken}, nil
	case "apple":
		var request model.AppleMobileLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return nil, errors.New("잘못된 요청 본문입니다.")
		}
		if strings.TrimSpace(request.ChallengeID) == "" ||
			strings.TrimSpace(request.IdentityToken) == "" ||
			strings.TrimSpace(request.AuthorizationCode) == "" {
			return nil, errors.New("Apple credential is incomplete")
		}
		return model.AppleAuthorization{
			ChallengeID:       request.ChallengeID,
			IdentityToken:     request.IdentityToken,
			AuthorizationCode: request.AuthorizationCode,
			GivenName:         strings.TrimSpace(request.GivenName),
			FamilyName:        strings.TrimSpace(request.FamilyName),
		}, nil
	default:
		return nil, service.ErrInvalidSocialProvider
	}
}

func (h *AuthHandler) DisconnectSocial(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	result, err := h.service.Disconnect(user.USRSeq, chi.URLParam(r, "provider"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidSocialProvider):
			respondError(w, http.StatusBadRequest, "INVALID_PROVIDER", "지원하지 않는 소셜 로그인 수단입니다.")
		case errors.Is(err, service.ErrLastLoginMethod):
			respondError(w, http.StatusConflict, "LAST_LOGIN_METHOD", "마지막 로그인 수단은 해제할 수 없습니다.")
		default:
			h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("social connection disconnect failed")
			respondError(w, http.StatusInternalServerError, "DISCONNECT_FAILED", "소셜 로그인 연결을 해제할 수 없습니다.")
		}
		return
	}
	respondJSON(w, http.StatusOK, result)
}
