// auth_social_link_handler.go — HTTP handler for social account linking requests
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog/log"
)

type socialLinkRequest struct {
	LinkToken string `json:"linkToken"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

const maxSocialLinkRequestBytes = 64 << 10

// SocialLink completes the canonical mobile attach-only flow. The legacy
// phone-based merge payload is deliberately rejected because provider email
// and caller-supplied profile fields are not identity proof.
func (h *AuthHandler) SocialLink(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeSocialLinkRequest(w, r)
	if !ok {
		return
	}

	result, err := h.service.CompleteCanonicalSocialLink(req.LinkToken, req.Email, req.Password)
	if err != nil {
		respondSocialLinkError(w, err)
		return
	}
	writeMobileAuthResult(w, result)
}

func (h *AuthHandler) SocialLinkWeb(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeSocialLinkRequest(w, r)
	if !ok {
		return
	}
	user, err := h.service.CompleteCanonicalSocialLinkIdentity(req.LinkToken, req.Email, req.Password)
	if err != nil {
		respondSocialLinkError(w, err)
		return
	}
	if err := h.service.LoginWithBridge(user, w, r); err != nil {
		if errors.Is(err, service.ErrLoginSuspended) || errors.Is(err, service.ErrLoginWithdrawn) {
			respondError(w, http.StatusForbidden, service.LoginErrorCode(err), "이 계정은 현재 로그인할 수 없습니다.")
			return
		}
		log.Error().Msg("canonical web social link bridge failed")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 처리 중 오류가 발생했습니다")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeSocialLinkRequest(w http.ResponseWriter, r *http.Request) (socialLinkRequest, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxSocialLinkRequestBytes+1))
	if err != nil || len(body) > maxSocialLinkRequestBytes {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return socialLinkRequest{}, false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return socialLinkRequest{}, false
	}
	legacyKeys := []string{"token", "mode", "phone", "name", "fn", "fmDept", "profileImageUrl"}
	for _, key := range legacyKeys {
		if _, found := fields[key]; found {
			respondError(w, http.StatusGone, "LEGACY_LINK_FLOW_DISABLED", "지원이 종료된 계정 연동 요청입니다")
			return socialLinkRequest{}, false
		}
	}
	allowedKeys := map[string]struct{}{"linkToken": {}, "email": {}, "password": {}}
	for key := range fields {
		if _, allowed := allowedKeys[key]; !allowed {
			respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
			return socialLinkRequest{}, false
		}
	}
	var req socialLinkRequest
	if len(fields) != len(allowedKeys) || json.Unmarshal(body, &req) != nil ||
		strings.TrimSpace(req.LinkToken) == "" || strings.TrimSpace(req.Email) == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Missing required fields")
		return socialLinkRequest{}, false
	}
	return req, true
}

func respondSocialLinkError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrSocialLinkTokenInvalid):
		respondError(w, http.StatusBadRequest, "INVALID_TOKEN", "Link token expired or invalid")
	case errors.Is(err, repository.ErrSocialLinkTokenConsumed):
		respondError(w, http.StatusConflict, "TOKEN_ALREADY_USED", "이미 처리된 소셜 링크 토큰입니다")
	case errors.Is(err, repository.ErrSocialLinkReauthLocked):
		respondError(w, http.StatusTooManyRequests, "REAUTHENTICATION_LOCKED", "새 계정 연동 요청을 시작해주세요")
	case errors.Is(err, repository.ErrSocialLinkReauth):
		respondError(w, http.StatusUnauthorized, "REAUTHENTICATION_REQUIRED", "이메일과 비밀번호를 다시 확인해주세요")
	case errors.Is(err, repository.ErrSocialIdentityOwner):
		respondError(w, http.StatusConflict, "ACCOUNT_MERGE_NOT_SUPPORTED", "이미 다른 회원에게 연결된 계정입니다")
	default:
		log.Error().Msg("canonical social link failed")
		respondError(w, http.StatusInternalServerError, "LINK_FAILED", "계정 연동에 실패했습니다")
	}
}
