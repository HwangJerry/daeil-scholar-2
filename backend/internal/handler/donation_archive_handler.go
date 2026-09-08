package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

type DonationArchiveStore interface {
	List(context.Context, int64) ([]repository.DonationArchiveMetadata, error)
	Read(context.Context, int64, int, string, func([]byte) ([]byte, error)) (json.RawMessage, error)
}
type DonationArchiveHandler struct {
	Store DonationArchiveStore
	Open  func([]byte) ([]byte, error)
}

func archiveOperator(w http.ResponseWriter, r *http.Request) *model.AuthUser {
	w.Header().Set("Cache-Control", "no-store")
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, 401, "UNAUTHORIZED", "로그인이 필요합니다.")
		return nil
	}
	if user.AdminRole == nil || *user.AdminRole != model.AdminRoleRoot {
		respondError(w, 403, "ADMIN_ROLE_REQUIRED", "최고 관리자 권한이 필요합니다.")
		return nil
	}
	return user
}
func (h *DonationArchiveHandler) List(w http.ResponseWriter, r *http.Request) {
	if archiveOperator(w, r) == nil {
		return
	}
	var before int64
	if raw := r.URL.Query().Get("before"); raw != "" {
		var err error
		before, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || before < 0 {
			respondError(w, 400, "INVALID_CURSOR", "잘못된 페이지입니다.")
			return
		}
	}
	items, err := h.Store.List(r.Context(), before)
	if err != nil {
		respondError(w, 503, "ARCHIVE_UNAVAILABLE", "보관소를 조회할 수 없습니다. 서버 설정을 확인해주세요.")
		return
	}
	respondJSON(w, 200, map[string]interface{}{"items": items})
}
func (h *DonationArchiveHandler) Read(w http.ResponseWriter, r *http.Request) {
	user := archiveOperator(w, r)
	if user == nil {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req struct {
		Purpose string `json:"purpose"`
	}
	if err != nil || id <= 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&req) != nil {
		respondError(w, 400, "INVALID_ARCHIVE_REQUEST", "조회 요청을 확인해주세요.")
		return
	}
	switch req.Purpose {
	case "receipt_inquiry", "accounting_review", "official_request":
	default:
		respondError(w, 400, "PURPOSE_REQUIRED", "조회 목적을 선택해주세요.")
		return
	}
	if h.Open == nil {
		respondError(w, 503, "ARCHIVE_UNAVAILABLE", "보관소 암호화 키 설정을 확인해주세요.")
		return
	}
	data, err := h.Store.Read(r.Context(), id, user.USRSeq, req.Purpose, h.Open)
	if errors.Is(err, sql.ErrNoRows) {
		respondError(w, 404, "ARCHIVE_NOT_FOUND", "증빙이 없거나 보관 기한이 지났습니다.")
		return
	}
	if err != nil {
		respondError(w, 503, "ARCHIVE_UNAVAILABLE", "증빙 복호화 또는 열람 기록 저장에 실패했습니다.")
		return
	}
	respondJSON(w, 200, data)
}
