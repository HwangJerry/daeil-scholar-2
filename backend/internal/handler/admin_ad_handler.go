// Admin ad handler — HTTP lifecycle for ad management endpoints
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminAdHandler struct {
	service *service.AdminAdService
}

func NewAdminAdHandler(svc *service.AdminAdService) *AdminAdHandler {
	return &AdminAdHandler{service: svc}
}

func (h *AdminAdHandler) List(w http.ResponseWriter, r *http.Request) {
	ads, err := h.service.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to list ads")
		return
	}
	respondJSON(w, http.StatusOK, ads)
}

func (h *AdminAdHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.AdminAdInsert
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	seq, err := h.service.Create(&req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create ad")
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int{"maSeq": seq})
}

func (h *AdminAdHandler) Update(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req model.AdminAdInsert
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.Update(seq, &req); err != nil {
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update ad")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminAdHandler) Delete(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	if err := h.service.Delete(seq); err != nil {
		respondError(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete ad")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminAdHandler) Stats(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(r.URL.Query().Get("maSeq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Missing maSeq")
		return
	}
	stats, err := h.service.GetStats(seq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STATS_FAILED", "Failed to get ad stats")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}
