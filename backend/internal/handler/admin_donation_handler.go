// Admin donation handler — HTTP lifecycle for donation config endpoints
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminDonationHandler struct {
	service *service.DonationConfigOrchestrator
}

func NewAdminDonationHandler(svc *service.DonationConfigOrchestrator) *AdminDonationHandler {
	return &AdminDonationHandler{service: svc}
}

func (h *AdminDonationHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.service.GetConfig()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "CONFIG_FAILED", "Failed to get donation config")
		return
	}
	respondJSON(w, http.StatusOK, cfg)
}

type updateDonationConfigRequest struct {
	Goal      int64  `json:"goal"`
	ManualAdj int64  `json:"manualAdj"`
	Note      string `json:"note"`
	Overwrite bool   `json:"overwrite"`
}

func (h *AdminDonationHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req updateDonationConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	if err := h.service.UpdateConfig(req.Goal, req.ManualAdj, req.Note, req.Overwrite, user.USRSeq); err != nil {
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update config")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminDonationHandler) History(w http.ResponseWriter, r *http.Request) {
	days := parseIntParam(r.URL.Query().Get("days"))
	history, err := h.service.GetHistory(days)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "HISTORY_FAILED", "Failed to get history")
		return
	}
	respondJSON(w, http.StatusOK, history)
}

func (h *AdminDonationHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, total, err := h.service.ListOrders(
		parseIntParam(q.Get("page")),
		parseIntParam(q.Get("size")),
		q.Get("name"), q.Get("status"), q.Get("payType"),
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to list donation orders")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": rows, "total": total})
}

type updateDonationOrderRequest struct {
	Payment string `json:"payment"`
	Amount  int    `json:"amount"`
}

func (h *AdminDonationHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req updateDonationOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.UpdateOrder(seq, req.Payment, req.Amount); err != nil {
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update donation order")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
