// social_link_photo_handler.go — Pre-signup profile photo upload, gated by a valid social-link token (no DB write).
package handler

import (
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

const socialLinkPhotoMaxBytes = 5 << 20 // 5 MB

type SocialLinkPhotoHandler struct {
	uploader   *service.UploadOrchestrator
	linkTokens *service.SocialLinkTokenStore
	logger     zerolog.Logger
}

func NewSocialLinkPhotoHandler(uploader *service.UploadOrchestrator, linkTokens *service.SocialLinkTokenStore, logger zerolog.Logger) *SocialLinkPhotoHandler {
	return &SocialLinkPhotoHandler{uploader: uploader, linkTokens: linkTokens, logger: logger}
}

// Upload accepts a profile photo during the social signup flow before a member row exists.
// Auth is the cached link token; on success the cached SocialLinkData.ProfileImageURL is
// updated so the eventual /api/auth/social/link submit picks up the new URL by default.
func (h *SocialLinkPhotoHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, socialLinkPhotoMaxBytes)
	if err := r.ParseMultipartForm(socialLinkPhotoMaxBytes); err != nil {
		respondError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "File exceeds 5MB limit")
		return
	}
	token := r.FormValue("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "MISSING_TOKEN", "token 파라미터가 필요합니다")
		return
	}
	_, err := h.linkTokens.Snapshot(token)
	if errors.Is(err, service.ErrSocialLinkTokenConsumed) {
		respondError(w, http.StatusConflict, "TOKEN_ALREADY_USED", "이미 처리된 소셜 링크 토큰입니다")
		return
	}
	if err != nil {
		respondError(w, http.StatusNotFound, "INVALID_TOKEN", "유효한 소셜 링크 토큰이 아닙니다")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "NO_FILE", "No file provided")
		return
	}
	defer file.Close()

	result, err := h.uploader.Upload(file, header, "profile")
	if err != nil {
		h.logger.Error().Err(err).Msg("social link photo: upload failed")
		respondError(w, http.StatusInternalServerError, "UPLOAD_FAILED", "Photo upload failed")
		return
	}

	_, err = h.linkTokens.Update(token, func(data model.SocialLinkData) model.SocialLinkData {
		data.ProfileImageURL = result.URL
		return data
	})
	if errors.Is(err, service.ErrSocialLinkTokenInProgress) {
		respondError(w, http.StatusConflict, "TOKEN_IN_PROGRESS", "계정 연결이 처리 중입니다")
		return
	}
	if err != nil {
		respondError(w, http.StatusConflict, "INVALID_TOKEN", "유효한 소셜 링크 토큰이 아닙니다")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"url": result.URL})
}
