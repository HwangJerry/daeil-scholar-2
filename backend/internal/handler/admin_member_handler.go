// Admin member handler — HTTP lifecycle for member management endpoints
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminMemberHandler struct {
	service *service.AdminMemberService
}

func NewAdminMemberHandler(svc *service.AdminMemberService) *AdminMemberHandler {
	return &AdminMemberHandler{service: svc}
}

func (h *AdminMemberHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, total, err := h.service.List(
		parseIntParam(q.Get("page")),
		parseIntParam(q.Get("size")),
		q.Get("name"), q.Get("fn"), q.Get("status"),
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to list members")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": rows, "total": total})
}

func (h *AdminMemberHandler) Detail(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	detail, err := h.service.GetDetail(seq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DETAIL_FAILED", "Failed to load member")
		return
	}
	if detail == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Member not found")
		return
	}
	hasKakao, _ := h.service.HasKakaoLink(seq)
	respondJSON(w, http.StatusOK, map[string]interface{}{"member": detail, "kakaoLinked": hasKakao})
}

type updateMemberRequest struct {
	Status string `json:"status"`
}

func (h *AdminMemberHandler) Update(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req updateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.UpdateStatus(seq, req.Status); err != nil {
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminMemberHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetMemberStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STATS_FAILED", "Failed to get member stats")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}
