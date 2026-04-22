// auth_social_link_handler.go — HTTP handler for social account linking requests
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog/log"
)

type socialLinkRequest struct {
	Token          string   `json:"token"`
	Mode           string   `json:"mode"` // "new" (default) | "merge"
	Name           string   `json:"name"`
	Phone          string   `json:"phone"`
	Email          string   `json:"email"`
	FN             string   `json:"fn"`
	FmDept         string   `json:"fmDept"`
	JobCat         *int     `json:"jobCat"`
	BizName        string   `json:"bizName"`
	BizDesc        string   `json:"bizDesc"`
	BizAddr        string   `json:"bizAddr"`
	Position       string   `json:"position"`
	Tags           []string `json:"tags"`
	USRPhonePublic string   `json:"usrPhonePublic"`
	USREmailPublic string   `json:"usrEmailPublic"`
}

// SocialLink handles the account linking HTTP flow for all social providers.
// Behavior is mode-driven: "new" creates a fresh member, "merge" attaches the
// social link to an existing member found by phone (user confirmed via UI banner).
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
	if req.Token == "" || req.Phone == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Missing required fields")
		return
	}
	if mode == service.SocialLinkModeNew && req.Name == "" {
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

	cached, found := h.cache.Get("social_link:" + req.Token)
	if !found {
		respondError(w, http.StatusBadRequest, "INVALID_TOKEN", "Link token expired or invalid")
		return
	}
	linkData, ok := cached.(model.SocialLinkData)
	if !ok {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Invalid cached data")
		return
	}
	h.cache.Delete("social_link:" + req.Token)

	// In merge mode the server uses existing member's email/name; the form fields are readonly.
	// Cached email is the authoritative reference for the social link row.
	linkEmail := req.Email
	if linkEmail == "" {
		linkEmail = linkData.Email
	}

	user, isNew, err := h.service.LinkSocialAccount(service.SocialLinkParams{
		Mode:            mode,
		Provider:        linkData.Provider,
		SocialID:        linkData.SocialID,
		Email:           linkEmail,
		Name:            req.Name,
		Phone:           req.Phone,
		FN:              req.FN,
		FmDept:          req.FmDept,
		JobCat:          req.JobCat,
		BizName:         req.BizName,
		BizDesc:         req.BizDesc,
		BizAddr:         req.BizAddr,
		Position:        req.Position,
		Tags:            req.Tags,
		USRPhonePublic:  req.USRPhonePublic,
		USREmailPublic:  req.USREmailPublic,
		ProfileImageURL: linkData.ProfileImageURL,
	}, h.memberSvc)
	if err != nil {
		log.Error().Err(err).Str("provider", linkData.Provider).Str("socialID", linkData.SocialID).Str("mode", string(mode)).Msg("social link failed")
		switch {
		case errors.Is(err, service.ErrPhoneAlreadyRegistered):
			respondError(w, http.StatusConflict, "PHONE_TAKEN", "이미 가입된 전화번호입니다. 통합 회원가입으로 진행해주세요.")
		case errors.Is(err, service.ErrPhoneNotFound):
			respondError(w, http.StatusConflict, "PHONE_NOT_MATCHED", "해당 전화번호의 기존 회원을 찾을 수 없습니다")
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

	if err := h.service.LoginWithBridge(user, w, r); err != nil {
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 처리 중 오류가 발생했습니다")
		return
	}
	if linkData.Provider == "KT" {
		h.service.CacheKakaoToken(user.USRSeq, linkData.AccessToken)
	}

	authUser := model.AuthUser{USRSeq: user.USRSeq, USRID: user.USRID, USRName: user.USRName, USRStatus: user.USRStatus}
	respondJSON(w, http.StatusOK, authUser)
}
