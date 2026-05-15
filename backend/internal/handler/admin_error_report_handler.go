package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/observability"
	"github.com/rs/zerolog"
)

// AdminErrorReportHandler accepts frontend JS error reports from the admin SPA
// and forwards them to the debug-agent via the zerolog hook.
type AdminErrorReportHandler struct {
	logger zerolog.Logger
	hook   *observability.Hook
}

func NewAdminErrorReportHandler(logger zerolog.Logger, hook *observability.Hook) *AdminErrorReportHandler {
	return &AdminErrorReportHandler{logger: logger, hook: hook}
}

type frontendErrorReport struct {
	Message    string `json:"message"`
	Stack      string `json:"stack"`
	URL        string `json:"url"`
	Component  string `json:"component"`
}

func (h *AdminErrorReportHandler) Report(w http.ResponseWriter, r *http.Request) {
	var body frontendErrorReport
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	msg := body.Message
	if msg == "" {
		msg = "frontend error"
	}

	h.hook.ReportPanic(msg, []byte(body.Stack), map[string]interface{}{
		"source":    "admin-spa",
		"url":       body.URL,
		"component": body.Component,
	})

	h.logger.Error().
		Str("source", "admin-spa").
		Str("url", body.URL).
		Str("component", body.Component).
		Str("stack", body.Stack).
		Msg(msg)

	w.WriteHeader(http.StatusNoContent)
}
