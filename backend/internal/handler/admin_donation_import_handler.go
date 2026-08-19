package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
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

	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "NO_FILE", "업로드할 엑셀 파일이 필요합니다.")
		return
	}
	defer file.Close()
	donationDate := strings.TrimSpace(r.FormValue("donationDate"))
	if donationDate == "" {
		respondError(w, http.StatusBadRequest, "NO_DONATION_DATE", "기부 반영일자가 필요합니다.")
		return
	}

	result, err := h.service.ParsePreview(file, donationDate)
	if err != nil {
		if errors.Is(err, service.ErrInvalidDonationDate) {
			respondError(w, http.StatusBadRequest, "INVALID_DONATION_DATE", err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidDonationImportFile) {
			respondError(w, http.StatusBadRequest, "INVALID_DONATION_FILE", err.Error())
			return
		}
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
		respondError(w, http.StatusInternalServerError, "DONATION_COMMIT_FAILED", "기부 내역을 반영하지 못했습니다.")
		return
	}
	respondJSON(w, http.StatusOK, result)
}
