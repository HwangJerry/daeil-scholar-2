package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AlumniHandler struct {
	service *service.AlumniService
}

func NewAlumniHandler(service *service.AlumniService) *AlumniHandler {
	return &AlumniHandler{service: service}
}

func (h *AlumniHandler) Search(w http.ResponseWriter, r *http.Request) {
	params := model.AlumniSearchParams{
		Name:       r.URL.Query().Get("name"),
		Cohort:     r.URL.Query().Get("cohort"),
		Department: r.URL.Query().Get("department"),
		JobRole:    r.URL.Query().Get("jobRole"),
	}
	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil {
		params.Page = page
	}
	if size, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil {
		params.Size = size
	}
	if graduationYear, err := strconv.Atoi(r.URL.Query().Get("graduationYear")); err == nil {
		params.GraduationYear = graduationYear
	}
	if jobCategory, err := strconv.Atoi(r.URL.Query().Get("jobCategory")); err == nil {
		params.JobCategory = jobCategory
	}
	params.MessageRecipientsOnly = r.URL.Query().Get("messageRecipientsOnly") == "true"
	query := r.URL.Query()
	// Plural keys are comma-separated; repeated legacy keys are also accepted.
	params.Cohorts = alumniQueryValues(query["cohorts"], query["cohort"])
	params.Departments = alumniQueryValues(query["departments"], query["department"])
	if query.Has("cohorts") {
		params.Cohort = ""
	}
	if query.Has("departments") {
		params.Department = ""
	}
	if query.Has("jobCategories") {
		params.JobCategory = 0
	}
	for _, value := range alumniQueryValues(query["jobCategories"], query["jobCategory"]) {
		if seq, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seq > 0 {
			params.JobCategories = append(params.JobCategories, seq)
		}
	}
	result, err := h.service.Search(params)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INVALID_REQUEST", "Failed to search alumni")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func alumniQueryValues(plural, legacy []string) []string {
	if plural == nil {
		return legacy
	}
	var values []string
	for _, value := range plural {
		values = append(values, strings.Split(value, ",")...)
	}
	return values
}

func (h *AlumniHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	userSeq, err := strconv.Atoi(chi.URLParam(r, "userSeq"))
	if err != nil || userSeq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_USER_SEQ", "회원 식별자가 올바르지 않습니다")
		return
	}
	viewer := middleware.GetAuthUser(r.Context())
	if viewer == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	detail, err := h.service.GetDetail(viewer.USRSeq, userSeq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INVALID_REQUEST", "동문 정보를 불러오지 못했습니다")
		return
	}
	if detail == nil {
		respondError(w, http.StatusNotFound, "INVALID_USER_SEQ", "동문 정보를 찾을 수 없습니다")
		return
	}
	respondJSON(w, http.StatusOK, detail)
}

// GetWidgetPreview handles GET /api/alumni/widget for approved authenticated alumni.
func (h *AlumniHandler) GetWidgetPreview(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetWidgetPreview()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INVALID_REQUEST", "Failed to load widget data")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *AlumniHandler) GetFilters(w http.ResponseWriter, r *http.Request) {
	filters, err := h.service.GetFilters()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INVALID_REQUEST", "Failed to load filters")
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
