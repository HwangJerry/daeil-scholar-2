// Admin notice handler — HTTP lifecycle for notice CRUD endpoints
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/presenter"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminNoticeHandler struct {
	service   *service.AdminNoticeService
	presenter *presenter.FeedPresenter
}

func NewAdminNoticeHandler(svc *service.AdminNoticeService, pres *presenter.FeedPresenter) *AdminNoticeHandler {
	return &AdminNoticeHandler{service: svc, presenter: pres}
}

func (h *AdminNoticeHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, total, err := h.service.List(parseIntParam(q.Get("page")), parseIntParam(q.Get("size")), q.Get("keyword"), parseIntParam(q.Get("category")))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to list notices")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": rows, "total": total})
}

func (h *AdminNoticeHandler) Detail(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	detail, err := h.service.GetForEdit(seq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DETAIL_FAILED", "Failed to load notice")
		return
	}
	if detail == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Notice not found")
		return
	}
	respondJSON(w, http.StatusOK, h.presenter.FormatNoticeDetailForAdmin(detail))
}

type createNoticeRequest struct {
	Subject          string `json:"subject"`
	ContentMd        string `json:"contentMd"`
	IsPinned         string `json:"isPinned"`
	AttachedFileSeqs []int  `json:"attachedFileSeqs"`
	// CategorySeq is optional: missing files a new post under the default
	// category and leaves an edited post's category unchanged.
	CategorySeq *int `json:"categorySeq"`
}

type setNoticeCategoryRequest struct {
	CategorySeq int `json:"categorySeq"`
}

// respondCategoryRuleError writes a 400 for a refused category and reports
// whether err was one.
func respondCategoryRuleError(w http.ResponseWriter, err error) bool {
	var rule *service.FeedCategoryRuleError
	if !errors.As(err, &rule) {
		return false
	}
	respondError(w, http.StatusBadRequest, rule.Code, rule.Message)
	return true
}

func (h *AdminNoticeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createNoticeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	seq, err := h.service.Create(req.Subject, req.ContentMd, user.USRName, user.USRSeq, req.IsPinned, req.AttachedFileSeqs, req.CategorySeq)
	if err != nil {
		if respondCategoryRuleError(w, err) {
			return
		}
		respondError(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create notice")
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int{"seq": seq})
}

func (h *AdminNoticeHandler) Update(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req createNoticeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.Update(seq, req.Subject, req.ContentMd, req.IsPinned, req.AttachedFileSeqs, req.CategorySeq); err != nil {
		if respondCategoryRuleError(w, err) {
			return
		}
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update notice")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetCategory handles PUT /api/admin/feed/{seq}/category — reclassify only.
func (h *AdminNoticeHandler) SetCategory(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req setNoticeCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CategorySeq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.SetCategory(seq, req.CategorySeq); err != nil {
		if respondCategoryRuleError(w, err) {
			return
		}
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update notice category")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminNoticeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	if err := h.service.Delete(seq); err != nil {
		respondError(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete notice")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminNoticeHandler) TogglePin(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	if err := h.service.TogglePin(seq); err != nil {
		respondError(w, http.StatusInternalServerError, "PIN_FAILED", "Failed to toggle pin")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
