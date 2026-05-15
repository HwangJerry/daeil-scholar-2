// social_link_photo_handler.go — Pre-signup profile photo upload, gated by a valid social-link token (no DB write).
package handler

import (
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

const socialLinkPhotoMaxBytes = 5 << 20 // 5 MB

type SocialLinkPhotoHandler struct {
	uploader *service.UploadOrchestrator
	cache    *cache.Cache
	logger   zerolog.Logger
}

func NewSocialLinkPhotoHandler(uploader *service.UploadOrchestrator, cacheStore *cache.Cache, logger zerolog.Logger) *SocialLinkPhotoHandler {
	return &SocialLinkPhotoHandler{uploader: uploader, cache: cacheStore, logger: logger}
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
	cached, found := h.cache.Get("social_link:" + token)
	if !found {
		respondError(w, http.StatusNotFound, "TOKEN_NOT_FOUND", "유효한 소셜 링크 토큰이 아닙니다")
		return
	}
	linkData, ok := cached.(model.SocialLinkData)
	if !ok {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Invalid cached data")
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

	linkData.ProfileImageURL = result.URL
	h.cache.Set("social_link:"+token, linkData, 5*time.Minute)

	respondJSON(w, http.StatusOK, map[string]string{"url": result.URL})
}
