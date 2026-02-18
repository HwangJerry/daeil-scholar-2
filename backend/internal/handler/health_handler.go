package handler

import (
	"net/http"

	"github.com/jmoiron/sqlx"
)

type HealthHandler struct {
	db *sqlx.DB
}

func NewHealthHandler(db *sqlx.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	status := map[string]string{"status": "ok"}
	if err := h.db.PingContext(r.Context()); err != nil {
		status["status"] = "degraded"
		status["db"] = "unreachable"
		respondJSON(w, http.StatusServiceUnavailable, status)
		return
	}
	respondJSON(w, http.StatusOK, status)
}
