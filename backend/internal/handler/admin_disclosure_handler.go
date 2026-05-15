// Admin disclosure handler — HTTP lifecycle for public-disclosure CRUD endpoints
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/presenter"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminDisclosureHandler struct {
	service   *service.AdminDisclosureService
	presenter *presenter.FeedPresenter
}

func NewAdminDisclosureHandler(svc *service.AdminDisclosureService, pres *presenter.FeedPresenter) *AdminDisclosureHandler {
	return &AdminDisclosureHandler{service: svc, presenter: pres}
}

func (h *AdminDisclosureHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, total, err := h.service.List(parseIntParam(q.Get("page")), parseIntParam(q.Get("size")), q.Get("keyword"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to list disclosures")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": rows, "total": total})
}

func (h *AdminDisclosureHandler) Detail(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	detail, err := h.service.GetForEdit(seq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DETAIL_FAILED", "Failed to load disclosure")
		return
	}
	if detail == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Disclosure not found")
		return
	}
	respondJSON(w, http.StatusOK, h.presenter.FormatNoticeDetailForAdmin(detail))
}

type createDisclosureRequest struct {
	Subject          string `json:"subject"`
	ContentMd        string `json:"contentMd"`
	AttachedFileSeqs []int  `json:"attachedFileSeqs"`
}

func (h *AdminDisclosureHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createDisclosureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	seq, err := h.service.Create(req.Subject, req.ContentMd, user.USRName, user.USRSeq, req.AttachedFileSeqs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create disclosure")
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int{"seq": seq})
}

func (h *AdminDisclosureHandler) Update(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req createDisclosureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.Update(seq, req.Subject, req.ContentMd, req.AttachedFileSeqs); err != nil {
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update disclosure")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminDisclosureHandler) Delete(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	if err := h.service.Delete(seq); err != nil {
		respondError(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete disclosure")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
