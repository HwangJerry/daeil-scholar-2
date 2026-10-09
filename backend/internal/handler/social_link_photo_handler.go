// social_link_photo_handler.go — Pre-signup profile photo upload, gated by a valid social-link token (no DB write).
package handler

import (
	"errors"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

const socialLinkPhotoMaxBytes = 5 << 20 // 5 MB

type socialSignupPhotoUploader interface {
	Upload(multipart.File, *multipart.FileHeader, string) (*service.UploadResult, error)
	DiscardUnclaimedProfile(*service.UploadResult) error
}

type SocialLinkPhotoHandler struct {
	uploader   socialSignupPhotoUploader
	linkTokens *service.SocialLinkTokenStore
	logger     zerolog.Logger
}

func NewSocialLinkPhotoHandler(uploader socialSignupPhotoUploader, linkTokens *service.SocialLinkTokenStore, logger zerolog.Logger) *SocialLinkPhotoHandler {
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
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
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

	err = h.linkTokens.AttachUpload(token, result)
	if err != nil {
		h.discardUnusedUpload(result)
	}

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

// Upload may finish after cancel/expiry/Begin. Discard only this actual result,
// re-checking persistent ownership on each bounded retry.
func (h *SocialLinkPhotoHandler) discardUnusedUpload(result *service.UploadResult) {
	if err := h.uploader.DiscardUnclaimedProfile(result); err == nil {
		return
	}
	h.logger.Error().Int("fSeq", result.FSeq).Msg("social signup photo discard retry scheduled")
	copyResult := *result
	go func() {
		for _, delay := range []time.Duration{time.Second, 5 * time.Second, 30 * time.Second} {
			time.Sleep(delay)
			if err := h.uploader.DiscardUnclaimedProfile(&copyResult); err == nil {
				return
			} else {
				h.logger.Error().Err(err).Int("fSeq", copyResult.FSeq).Msg("social signup photo discard failed")
			}
		}
	}()
}
