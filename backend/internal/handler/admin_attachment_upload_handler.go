// Admin attachment upload handler — HTTP lifecycle for notice attachment uploads
package handler

import (
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/service"
)

type AdminAttachmentUploadHandler struct {
	orchestrator *service.AttachmentUploadOrchestrator
	maxSizeMB    int
}

func NewAdminAttachmentUploadHandler(orchestrator *service.AttachmentUploadOrchestrator, maxSizeMB int) *AdminAttachmentUploadHandler {
	return &AdminAttachmentUploadHandler{orchestrator: orchestrator, maxSizeMB: maxSizeMB}
}

func (h *AdminAttachmentUploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	maxBytes := int64(h.maxSizeMB) << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		respondError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "File exceeds size limit")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "NO_FILE", "No file provided")
		return
	}
	defer file.Close()

	result, err := h.orchestrator.Upload(file, header, "notice-attachment")
	if err != nil {
		if strings.HasPrefix(err.Error(), "blocked file type") {
			respondError(w, http.StatusBadRequest, "BLOCKED_EXT", err.Error())
			return
		}
		if strings.HasPrefix(err.Error(), "unsupported file type") {
			respondError(w, http.StatusBadRequest, "UNSUPPORTED_EXT", err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "UPLOAD_FAILED", "File upload failed")
		return
	}
	respondJSON(w, http.StatusOK, result)
}
