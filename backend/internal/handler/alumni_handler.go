package handler

import (
	"net/http"
	"strconv"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

type AlumniHandler struct {
	service *service.AlumniService
}

func NewAlumniHandler(service *service.AlumniService) *AlumniHandler {
	return &AlumniHandler{service: service}
}

func (h *AlumniHandler) Search(w http.ResponseWriter, r *http.Request) {
	params := model.AlumniSearchParams{
		FN:      r.URL.Query().Get("fn"),
		Dept:    r.URL.Query().Get("dept"),
		Name:    r.URL.Query().Get("name"),
		Company: r.URL.Query().Get("company"),
	}
	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil {
		params.Page = page
	}
	if size, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil {
		params.Size = size
	}
	if jobCat, err := strconv.Atoi(r.URL.Query().Get("jobCat")); err == nil {
		params.JobCat = jobCat
	}
	result, err := h.service.Search(params)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "SEARCH_FAILED", "Failed to search alumni")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// GetWidgetPreview handles GET /api/alumni/widget — public endpoint, no auth required.
func (h *AlumniHandler) GetWidgetPreview(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetWidgetPreview()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "WIDGET_FAILED", "Failed to load widget data")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *AlumniHandler) GetFilters(w http.ResponseWriter, r *http.Request) {
	filters, err := h.service.GetFilters()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "FILTER_FAILED", "Failed to load filters")
		return
	}
	respondJSON(w, http.StatusOK, filters)
}

// GetJobCategories handles GET /api/public/job-categories — public endpoint, no auth required.
func (h *AlumniHandler) GetJobCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.service.GetJobCategories()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "JOB_CATEGORY_FAILED", "Failed to load job categories")
		return
	}
	respondJSON(w, http.StatusOK, cats)
}
