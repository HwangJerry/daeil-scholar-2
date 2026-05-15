// history_handler.go — Public and admin HTTP handlers for history entries
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type HistoryHandler struct {
	svc *service.HistoryService
}

func NewHistoryHandler(svc *service.HistoryService) *HistoryHandler {
	return &HistoryHandler{svc: svc}
}

// GetGrouped handles GET /api/history — public, returns entries grouped by year.
func (h *HistoryHandler) GetGrouped(w http.ResponseWriter, r *http.Request) {
	groups, err := h.svc.GetGrouped()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "FETCH_FAILED", "Failed to load history")
		return
	}
	respondJSON(w, http.StatusOK, groups)
}

// AdminList handles GET /api/admin/history — flat list for admin table.
func (h *HistoryHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	entries, err := h.svc.GetAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to list history entries")
		return
	}
	respondJSON(w, http.StatusOK, entries)
}

// AdminCreate handles POST /api/admin/history.
func (h *HistoryHandler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var req model.HistoryUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	id, err := h.svc.Create(req)
	if err != nil {
		var ve *model.ValidationError
		if errors.As(err, &ve) {
			respondError(w, http.StatusBadRequest, "VALIDATION_FAILED", ve.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create history entry")
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int64{"seq": id})
}

// AdminUpdate handles PUT /api/admin/history/{seq}.
func (h *HistoryHandler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req model.HistoryUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.svc.Update(seq, req); err != nil {
		var ve *model.ValidationError
		if errors.As(err, &ve) {
			respondError(w, http.StatusBadRequest, "VALIDATION_FAILED", ve.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update history entry")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AdminDelete handles DELETE /api/admin/history/{seq}.
func (h *HistoryHandler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	if err := h.svc.Delete(seq); err != nil {
		respondError(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete history entry")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
