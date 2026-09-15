// comment_report_handler.go — User reporting and admin-only moderation endpoints.
package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

type CommentReportHandler struct{ Service *service.CommentReportService }

func commentReportError(w http.ResponseWriter, err error) {
	var invalid *model.ValidationError
	switch {
	case errors.As(err, &invalid):
		respondError(w, http.StatusBadRequest, "INVALID_REPORT", invalid.Error())
	case errors.Is(err, sql.ErrNoRows):
		respondError(w, http.StatusNotFound, "COMMENT_NOT_REPORTABLE", "현재 신고할 수 없는 댓글입니다.")
	case errors.Is(err, repository.ErrCommentReportResolved):
		respondError(w, http.StatusConflict, "REPORT_ALREADY_RESOLVED", "이미 다른 결과로 처리된 신고입니다.")
	default:
		respondError(w, http.StatusInternalServerError, "REPORT_FAILED", "신고를 처리하지 못했습니다. 다시 시도해주세요.")
	}
}

func (h *CommentReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.")
		return
	}
	var request model.CommentReportRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REPORT", "신고 내용을 확인해주세요.")
		return
	}
	request.PostID, _ = strconv.ParseInt(chi.URLParam(r, "seq"), 10, 64)
	request.CommentID, _ = strconv.ParseInt(chi.URLParam(r, "cSeq"), 10, 64)
	id, err := h.Service.Create(user.USRSeq, request)
	if err != nil {
		commentReportError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"reportId": id, "status": "received"})
}

func (h *CommentReportHandler) List(w http.ResponseWriter, r *http.Request) {
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
		commentReportError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *CommentReportHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var request model.CommentReportResolution
	if err != nil || json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&request) != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REPORT", "처리 내용을 확인해주세요.")
		return
	}
	if err = h.Service.Resolve(id, user.USRSeq, request); err != nil {
		commentReportError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
