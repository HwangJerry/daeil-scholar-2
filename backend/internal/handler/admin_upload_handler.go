// Admin upload handler — HTTP lifecycle for image upload endpoint
package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/service"
)

type AdminUploadHandler struct {
	orchestrator *service.UploadOrchestrator
	maxSizeMB    int
}

func NewAdminUploadHandler(orchestrator *service.UploadOrchestrator, maxSizeMB int) *AdminUploadHandler {
	return &AdminUploadHandler{orchestrator: orchestrator, maxSizeMB: maxSizeMB}
}

func (h *AdminUploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
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

	gate := r.URL.Query().Get("type")
	if gate == "" {
		gate = "notice"
	}
	allowedGates := map[string]bool{"notice": true, "ad": true, "profile": true}
	if !allowedGates[gate] {
		respondError(w, http.StatusBadRequest, "INVALID_TYPE", "Allowed types: notice, ad, profile")
		return
	}

	result, err := h.orchestrator.Upload(file, header, gate)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "UPLOAD_FAILED", "File upload failed")
		return
	}
	respondJSON(w, http.StatusOK, result)
}
