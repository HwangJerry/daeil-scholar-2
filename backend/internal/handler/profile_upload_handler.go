// profile_upload_handler.go — HTTP handlers for profile photo and biz card image uploads
package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
)

const profileUploadMaxBytes = 5 << 20 // 5 MB

type ProfileUploadHandler struct {
	uploadService *service.ProfileUploadService
}

func NewProfileUploadHandler(uploadSvc *service.ProfileUploadService) *ProfileUploadHandler {
	return &ProfileUploadHandler{uploadService: uploadSvc}
}

func (h *ProfileUploadHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, profileUploadMaxBytes)
	if err := r.ParseMultipartForm(profileUploadMaxBytes); err != nil {
		respondError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "File exceeds 5MB limit")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "NO_FILE", "No file provided")
		return
	}
	defer file.Close()

	url, err := h.uploadService.UploadProfilePhoto(user.USRSeq, file, header)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "UPLOAD_FAILED", "Photo upload failed")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"url": url})
}

func (h *ProfileUploadHandler) UploadBizCard(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, profileUploadMaxBytes)
	if err := r.ParseMultipartForm(profileUploadMaxBytes); err != nil {
		respondError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "File exceeds 5MB limit")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "NO_FILE", "No file provided")
		return
	}
	defer file.Close()

	url, err := h.uploadService.UploadBizCard(user.USRSeq, file, header)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "UPLOAD_FAILED", "Biz card upload failed")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"url": url})
}
