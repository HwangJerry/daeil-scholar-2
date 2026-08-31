// auth_social_link_handler.go — HTTP handler for social account linking requests
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog/log"
)

type socialLinkRequest struct {
	Token           string   `json:"token"`
	Mode            string   `json:"mode"`   // "new" (default)
	Client          string   `json:"client"` // "web" (default) | "mobile"
	Name            string   `json:"name"`
	Phone           string   `json:"phone"`
	Email           string   `json:"email"`
	FN              string   `json:"fn"`
	FmDept          string   `json:"fmDept"`
	JobCat          *int     `json:"jobCat"`
	BizName         string   `json:"bizName"`
	BizDesc         string   `json:"bizDesc"`
	BizAddr         string   `json:"bizAddr"`
	Position        string   `json:"position"`
	Tags            []string `json:"tags"`
	USRPhonePublic  string   `json:"usrPhonePublic"`
	USREmailPublic  string   `json:"usrEmailPublic"`
	ProfileImageURL *string  `json:"profileImageUrl,omitempty"`
}

// SocialLink creates a new member from a verified social-provider identity.
func (h *AuthHandler) SocialLink(w http.ResponseWriter, r *http.Request) {
	var req socialLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	mode := service.SocialLinkMode(req.Mode)
	if mode == "" {
		mode = service.SocialLinkModeNew
	}
	if mode != service.SocialLinkModeNew {
		respondError(w, http.StatusBadRequest, "UNSUPPORTED_SOCIAL_LINK_MODE", "지원하지 않는 소셜 계정 연결 모드입니다.")
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = model.NormalizePhoneNumber(req.Phone).String()
	req.FN = strings.TrimSpace(req.FN)
	req.FmDept = strings.TrimSpace(req.FmDept)
	if req.Token == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Missing required fields")
		return
	}
	if req.Phone == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Missing required fields")
		return
	}
	if req.Name == "" || req.Email == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Missing required fields")
		return
	}
	if req.FN == "" || req.FmDept == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "필수 입력값이 누락되었습니다")
		return
	}
	if !fnDigitRegex.MatchString(req.FN) {
		respondError(w, http.StatusBadRequest, "INVALID_FN", "기수는 숫자로 입력해주세요")
		return
	}
	if !model.IsValidDepartment(req.FmDept) {
		respondError(w, http.StatusBadRequest, "INVALID_DEPARTMENT", "유효하지 않은 학과입니다")
		return
	}
	if err := service.ValidateTags(req.Tags); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_TAG", "태그에 공백을 포함할 수 없습니다")
		return
	}

	lease, err := h.socialLinkTokens.Begin(req.Token)
	switch {
	case errors.Is(err, service.ErrSocialLinkTokenInProgress):
		respondError(w, http.StatusConflict, "TOKEN_IN_PROGRESS", "동일한 계정 연결 요청이 처리 중입니다.")
		return
	case errors.Is(err, service.ErrSocialLinkTokenConsumed):
		respondError(w, http.StatusConflict, "TOKEN_ALREADY_USED", "이미 처리된 소셜 링크 토큰입니다. 다시 소셜 로그인해주세요.")
		return
	case err != nil:
		respondError(w, http.StatusBadRequest, "INVALID_TOKEN", "Link token expired or invalid")
		return
	}
	tokenConsumed := false
	defer func() {
		if !tokenConsumed {
			_ = h.socialLinkTokens.Release(lease)
		}
	}()
	linkData := lease.Data
	// Provider email is profile metadata only. Keep the verifier-derived value on
	// the social link; use the form email only when the provider supplied none.
	linkEmail := linkData.Email
	if linkEmail == "" {
		linkEmail = req.Email
	}

	// Profile image: client may explicitly override the cached provider URL (replace or remove).
	// Nil pointer ⇒ field unset ⇒ keep the cached provider URL.
	// Non-nil ⇒ honor the client's value verbatim (including empty string for "no image").
	profileImageURL := linkData.ProfileImageURL
	if req.ProfileImageURL != nil {
		profileImageURL = *req.ProfileImageURL
	}

	provider := model.SocialProvider(linkData.Provider)
	credential := linkData.AccessToken
	if provider == model.SocialProviderApple {
		credential = linkData.RevocationToken
	}
	if err := h.socialLifecycle.EnsureCredentialStorageAvailable(credential); err != nil {
		h.logger.Error().Err(err).Str("provider", linkData.Provider).Msg("social credential storage unavailable")
		respondError(w, http.StatusServiceUnavailable, "CREDENTIAL_STORAGE_UNAVAILABLE", "소셜 계정 연결을 완료할 수 없습니다.")
		return
	}
	encryptedCredential, err := h.socialLifecycle.EncryptCredential(credential)
	if err != nil {
		h.logger.Error().Err(err).Str("provider", linkData.Provider).Msg("social credential encryption failed")
		respondError(w, http.StatusServiceUnavailable, "CREDENTIAL_STORAGE_UNAVAILABLE", "소셜 계정 연결을 완료할 수 없습니다.")
		return
	}
	user, isNew, err := h.service.LinkSocialAccount(service.SocialLinkParams{
		Mode:                mode,
		Provider:            linkData.Provider,
		SocialID:            linkData.SocialID,
		Email:               linkEmail,
		Name:                req.Name,
		Phone:               req.Phone,
		FN:                  req.FN,
		FmDept:              req.FmDept,
		JobCat:              req.JobCat,
		BizName:             req.BizName,
		BizDesc:             req.BizDesc,
		BizAddr:             req.BizAddr,
		Position:            req.Position,
		Tags:                req.Tags,
		USRPhonePublic:      req.USRPhonePublic,
		USREmailPublic:      req.USREmailPublic,
		ProfileImageURL:     profileImageURL,
		EncryptedCredential: encryptedCredential,
	}, h.memberSvc)
	if err != nil {
		log.Error().Err(err).Str("provider", linkData.Provider).Str("mode", string(mode)).Msg("social link failed")
		switch {
		case errors.Is(err, service.ErrInvalidPhone):
			respondError(w, http.StatusBadRequest, "INVALID_PHONE", "유효한 전화번호를 입력해주세요")
		case errors.Is(err, service.ErrPhoneAlreadyRegistered):
			respondError(w, http.StatusConflict, "PHONE_TAKEN", "이미 가입된 전화번호입니다. 기존 계정으로 로그인해주세요.")
		case errors.Is(err, service.ErrSocialAccountAlreadyLinked):
			respondError(w, http.StatusConflict, "SOCIAL_ACCOUNT_ALREADY_LINKED", "이미 다른 계정에 연결된 소셜 계정입니다. 다시 로그인해주세요.")
		case isLoginPolicyError(err):
			respondError(w, http.StatusForbidden, service.LoginErrorCode(err), "이 계정은 현재 로그인할 수 없습니다.")
		default:
			respondError(w, http.StatusInternalServerError, "LINK_FAILED", "계정 연동에 실패했습니다")
		}
		return
	}

	if req.Tags != nil {
		if saveErr := h.registerSvc.SaveInitialTags(user.USRSeq, req.Tags); saveErr != nil {
			if errors.Is(saveErr, service.ErrTagContainsWhitespace) {
				respondError(w, http.StatusBadRequest, "INVALID_TAG", "태그에 공백을 포함할 수 없습니다")
				return
			}
			log.Warn().Err(saveErr).Int("usrSeq", user.USRSeq).Bool("isNew", isNew).Msg("social link: failed to save tags")
		}
	}

	tokenConsumed = true
	if err := h.socialLinkTokens.Consume(lease); err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Str("provider", linkData.Provider).Msg("social link token consume failed")
		respondError(w, http.StatusInternalServerError, "LINK_STATE_FAILED", "계정 연결은 완료되었지만 상태를 확정할 수 없습니다. 다시 소셜 로그인해주세요.")
		return
	}
	authUser := model.AuthUser{USRSeq: user.USRSeq, USRID: user.USRID, USRName: user.USRName, USRStatus: user.USRStatus}
	if strings.EqualFold(req.Client, "mobile") {
		result, resultErr := h.socialAuth.CompleteMobileLink(user)
		if resultErr != nil {
			respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 토큰 발급에 실패했습니다")
			return
		}
		writeMobileAuthResult(w, result)
		return
	}

	if user.USRStatus == "BBB" {
		respondJSON(w, http.StatusAccepted, authUser)
		return
	}

	if err := h.service.LoginWithBridge(user, w, r); err != nil {
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 처리 중 오류가 발생했습니다")
		return
	}
	if linkData.Provider == "KT" {
		h.service.CacheKakaoToken(user.USRSeq, linkData.AccessToken)
	}

	respondJSON(w, http.StatusOK, authUser)
}
