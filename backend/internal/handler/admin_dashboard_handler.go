// Admin dashboard handler — HTTP lifecycle for dashboard endpoint
package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/service"
)

type AdminDashboardHandler struct {
	service *service.AdminDashboardService
}

func NewAdminDashboardHandler(svc *service.AdminDashboardService) *AdminDashboardHandler {
	return &AdminDashboardHandler{service: svc}
}

func (h *AdminDashboardHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DASHBOARD_FAILED", "Failed to load dashboard")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}
