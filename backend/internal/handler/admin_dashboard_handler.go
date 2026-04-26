// Admin dashboard handler — HTTP lifecycle for dashboard endpoint
package handler

import (
	"net/http"
	"time"

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

// ActiveUsers serves the DAU/MAU time series. Accepts ?from=YYYY-MM-DD&to=YYYY-MM-DD;
// defaults to the trailing 30 days ending today (KST).
func (h *AdminDashboardHandler) ActiveUsers(w http.ResponseWriter, r *http.Request) {
	today := service.Today()
	from := today.AddDate(0, 0, -29)
	to := today

	if v := r.URL.Query().Get("from"); v != "" {
		if d, err := time.ParseInLocation("2006-01-02", v, today.Location()); err == nil {
			from = d
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if d, err := time.ParseInLocation("2006-01-02", v, today.Location()); err == nil {
			to = d
		}
	}
	if to.Before(from) {
		respondError(w, http.StatusBadRequest, "INVALID_RANGE", "to must be on or after from")
		return
	}

	resp, err := h.service.ActiveUsers(from, to)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ACTIVE_USERS_FAILED", "Failed to load active users")
		return
	}
	respondJSON(w, http.StatusOK, resp)
}
