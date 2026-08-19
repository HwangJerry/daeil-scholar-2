package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog/log"
)

type AdminDonationImportHandler struct {
	service   *service.DonationImportService
	maxSizeMB int
}

func NewAdminDonationImportHandler(importService *service.DonationImportService, maxSizeMB int) *AdminDonationImportHandler {
	return &AdminDonationImportHandler{service: importService, maxSizeMB: maxSizeMB}
}

func (h *AdminDonationImportHandler) Preview(w http.ResponseWriter, r *http.Request) {
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
	donationDate := strings.TrimSpace(r.FormValue("donationDate"))
	if donationDate == "" {
		respondError(w, http.StatusBadRequest, "NO_DONATION_DATE", "기부 반영일자가 필요합니다.")
		return
	}

	result, err := h.service.ParsePreview(file, donationDate)
	if err != nil {
		var validationError *service.DonationImportFileValidationError
		if errors.As(err, &validationError) {
			respondJSON(w, http.StatusBadRequest, model.DonationImportErrorResponse{
				Code: "DONATION_FILE_VALIDATION_FAILED", Message: validationError.Error(), Errors: validationError.Errors,
			})
			return
		}
		if errors.Is(err, service.ErrInvalidDonationDate) {
			respondError(w, http.StatusBadRequest, "INVALID_DONATION_DATE", err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidDonationImportFile) {
			respondError(w, http.StatusBadRequest, "INVALID_DONATION_FILE", err.Error())
			return
		}
		log.Error().Err(err).Msg("donation import preview failed")
		respondError(w, http.StatusInternalServerError, "DONATION_PREVIEW_FAILED", "기부 내역 미리보기를 만들지 못했습니다.")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *AdminDonationImportHandler) Commit(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	var request model.DonationImportCommitRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "요청 본문은 기부 반영일자와 행 목록이어야 합니다.")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "요청 본문에는 하나의 JSON 값만 허용됩니다.")
		return
	}

	if strings.TrimSpace(request.DonationDate) == "" {
		respondError(w, http.StatusBadRequest, "NO_DONATION_DATE", "기부 반영일자가 필요합니다.")
		return
	}

	result, err := h.service.Commit(request.Rows, request.DonationDate, user.USRSeq, requestIP(r))
	if err != nil {
		if errors.Is(err, service.ErrInvalidDonationDate) {
			respondError(w, http.StatusBadRequest, "INVALID_DONATION_DATE", err.Error())
			return
		}
		var commitError *service.DonationImportCommitError
		if errors.As(err, &commitError) {
			respondJSON(w, http.StatusConflict, model.DonationImportErrorResponse{
				Code: "DONATION_IMPORT_COMMIT_REJECTED", Message: commitError.Error(), Errors: []model.DonationImportRowError{commitError.RowError},
			})
			return
		}
		log.Error().Err(err).Msg("donation import commit failed")
		respondError(w, http.StatusInternalServerError, "DONATION_COMMIT_FAILED", "기부 내역을 반영하지 못했습니다.")
		return
	}
	respondJSON(w, http.StatusOK, result)
}
