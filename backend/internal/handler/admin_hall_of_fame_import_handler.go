// admin_hall_of_fame_import_handler.go — Operator spreadsheet preview and commit endpoints.
package handler

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog/log"
)

type AdminHallOfFameImportHandler struct {
	service   *service.HallOfFameImportService
	maxSizeMB int
}

func NewAdminHallOfFameImportHandler(importService *service.HallOfFameImportService, maxSizeMB int) *AdminHallOfFameImportHandler {
	return &AdminHallOfFameImportHandler{service: importService, maxSizeMB: maxSizeMB}
}

func (h *AdminHallOfFameImportHandler) Preview(w http.ResponseWriter, r *http.Request) {
	actor, ok := hallOfFameActor(w, r)
	if !ok {
		return
	}
	maxBytes := int64(h.maxSizeMB) << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		respondError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "파일이 허용 크기를 초과했습니다.")
		return
	}
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "NO_FILE", "업로드할 엑셀 파일이 필요합니다.")
		return
	}
	defer file.Close()
	if !strings.EqualFold(filepath.Ext(fileHeader.Filename), ".xlsx") {
		respondError(w, http.StatusBadRequest, "INVALID_FILE_TYPE", ".xlsx 형식의 엑셀 파일만 업로드할 수 있습니다.")
		return
	}
	preview, err := h.service.Preview(file, actor.UsrSeq)
	if err != nil {
		if errors.Is(err, service.ErrInvalidHallOfFameFile) {
			respondError(w, http.StatusBadRequest, "INVALID_HALL_OF_FAME_FILE", err.Error())
			return
		}
		log.Error().Err(err).Msg("hall of fame import preview failed")
		respondError(w, http.StatusInternalServerError, "HALL_OF_FAME_PREVIEW_FAILED", "미리보기를 만들지 못했습니다.")
		return
	}
	respondJSON(w, http.StatusOK, preview)
}

func (h *AdminHallOfFameImportHandler) Commit(w http.ResponseWriter, r *http.Request) {
	actor, ok := hallOfFameActor(w, r)
	if !ok {
		return
	}
	var request model.HallOfFameImportCommitRequest
	if err := decodeClosedJSON(r, &request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "요청 본문이 올바르지 않습니다.")
		return
	}
	result, err := h.service.Commit(request.PreviewToken, request.RemoveIDs, actor)
	if err != nil {
		var rejection *service.HallOfFameImportRejection
		if errors.As(err, &rejection) {
			respondError(w, http.StatusConflict, "HALL_OF_FAME_IMPORT_REJECTED", rejection.Message)
			return
		}
		log.Error().Err(err).Msg("hall of fame import commit failed")
		respondError(w, http.StatusInternalServerError, "HALL_OF_FAME_COMMIT_FAILED", "반영하지 못했습니다.")
		return
	}
	respondJSON(w, http.StatusOK, result)
}
