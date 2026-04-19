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
func (h *AuthHandler) SocialLink(w http.ResponseWriter, r *http.Request) {
	var req socialLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.Token == "" || req.Name == "" || req.Phone == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Missing required fields")
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

	user, isNew, err := h.service.LinkSocialAccount(service.SocialLinkParams{
		Provider:       linkData.Provider,
		SocialID:       linkData.SocialID,
		Email:          req.Email,
		Name:           req.Name,
		Phone:          req.Phone,
		FN:             req.FN,
		FmDept:         req.FmDept,
		JobCat:         req.JobCat,
		BizName:        req.BizName,
		BizDesc:        req.BizDesc,
		BizAddr:        req.BizAddr,
		Position:       req.Position,
		Tags:           req.Tags,
		USRPhonePublic: req.USRPhonePublic,
		USREmailPublic: req.USREmailPublic,
	}, h.memberSvc)
	if isNew && len(req.Tags) > 0 {
		if saveErr := h.registerSvc.SaveInitialTags(user.USRSeq, req.Tags); saveErr != nil {
			log.Warn().Err(saveErr).Int("usrSeq", user.USRSeq).Msg("social link: failed to save initial tags")
		}
	}
	if err != nil {
		log.Error().Err(err).Str("provider", linkData.Provider).Str("socialID", linkData.SocialID).Msg("social link failed")
		if errors.Is(err, service.ErrNameMismatch) {
			respondError(w, http.StatusConflict, "NAME_MISMATCH", "전화번호는 등록되어 있으나 이름이 일치하지 않습니다")
			return
		}
		respondError(w, http.StatusInternalServerError, "LINK_FAILED", "계정 연동에 실패했습니다")
		return
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
