// admin_job_category_handler.go — HTTP lifecycle for job category admin endpoints
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminJobCategoryHandler struct {
	svc *service.AdminJobCategoryService
}

func NewAdminJobCategoryHandler(svc *service.AdminJobCategoryService) *AdminJobCategoryHandler {
	return &AdminJobCategoryHandler{svc: svc}
}

// List handles GET /api/admin/job-category — returns all categories (incl. hidden).
func (h *AdminJobCategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	cats, err := h.svc.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to list job categories")
		return
	}
	respondJSON(w, http.StatusOK, cats)
}

// Create handles POST /api/admin/job-category.
func (h *AdminJobCategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.AdminJobCategoryUpsert
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	id, err := h.svc.Create(req)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateCategoryName) {
			respondError(w, http.StatusConflict, "DUPLICATE_NAME", "이미 존재하는 카테고리 이름입니다.")
			return
		}
		respondError(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create job category")
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int64{"seq": id})
}

// Update handles PUT /api/admin/job-category/{seq}.
func (h *AdminJobCategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req model.AdminJobCategoryUpsert
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.svc.Update(seq, req); err != nil {
		if errors.Is(err, service.ErrDuplicateCategoryName) {
			respondError(w, http.StatusConflict, "DUPLICATE_NAME", "이미 존재하는 카테고리 이름입니다.")
			return
		}
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update job category")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Delete handles DELETE /api/admin/job-category/{seq} — soft delete (OPEN_YN='N').
func (h *AdminJobCategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	if err := h.svc.Delete(seq); err != nil {
		respondError(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete job category")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
