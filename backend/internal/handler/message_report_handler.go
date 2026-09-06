// message_report_handler.go — User reporting and admin-only moderation endpoints.
package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

type MessageReportHandler struct{ Service *service.MessageReportService }

func reportError(w http.ResponseWriter, err error) {
	var invalid *model.ValidationError
	switch {
	case errors.As(err, &invalid):
		respondError(w, http.StatusBadRequest, "INVALID_REPORT", invalid.Error())
	case errors.Is(err, sql.ErrNoRows):
		respondError(w, http.StatusNotFound, "REPORT_NOT_FOUND", "신고할 메시지가 없거나 이미 처리되었습니다.")
	default:
		respondError(w, http.StatusInternalServerError, "REPORT_FAILED", "신고를 처리하지 못했습니다. 다시 시도해주세요.")
	}
}

func (h *MessageReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.")
		return
	}
	var request model.MessageReportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REPORT", "신고 내용을 확인해주세요.")
		return
	}
	id, err := h.Service.Create(user.USRSeq, request)
	if err != nil {
		reportError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"reportId": id, "status": "received"})
}

func (h *MessageReportHandler) List(w http.ResponseWriter, r *http.Request) {
	before := int64(0)
	if raw := r.URL.Query().Get("before"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_CURSOR", "잘못된 페이지입니다.")
			return
		}
		before = value
	}
	items, err := h.Service.List(r.URL.Query().Get("status"), before)
	if err != nil {
		reportError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *MessageReportHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var request model.MessageReportResolution
	if err != nil || json.NewDecoder(r.Body).Decode(&request) != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REPORT", "처리 내용을 확인해주세요.")
		return
	}
	if err = h.Service.Resolve(id, user.USRSeq, request); err != nil {
		reportError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
