// admin_error_report_handler.go — Accept admin diagnostics without persisting raw input.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
)

// AdminErrorReportHandler accepts frontend JS error reports from the admin SPA
// and records only a fixed diagnostic label in the local server log.
type AdminErrorReportHandler struct {
	logger zerolog.Logger
}

func NewAdminErrorReportHandler(logger zerolog.Logger) *AdminErrorReportHandler {
	return &AdminErrorReportHandler{logger: logger}
}

type frontendErrorReport struct {
	Message   string `json:"message"`
	Stack     string `json:"stack"`
	URL       string `json:"url"`
	Component string `json:"component"`
}

func (h *AdminErrorReportHandler) Report(w http.ResponseWriter, r *http.Request) {
	var body frontendErrorReport
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.logger.Error().Str("source", "admin-spa").Msg("admin frontend error")

	w.WriteHeader(http.StatusNoContent)
}
