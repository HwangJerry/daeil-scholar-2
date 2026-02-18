package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/service"
)

type DonationHandler struct {
	service *service.DonationService
}

func NewDonationHandler(service *service.DonationService) *DonationHandler {
	return &DonationHandler{service: service}
}

func (h *DonationHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetSummary()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "SUMMARY_FAILED", "Failed to load summary")
		return
	}
	respondJSON(w, http.StatusOK, summary)
}
