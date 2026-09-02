// sentry_monitoring_handler.go — Admin HTTP proxy for Sentry summaries.
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

const (
	defaultSentryTopIssueLimit = 5
	maxSentryTopIssueLimit     = 20
)

type SentryMonitoringServicer interface {
	CrashSummary(ctx context.Context, topN int) (*model.SentryCrashSummaryResponse, error)
	PerformanceSummary(ctx context.Context) (*model.SentryPerformanceSummaryResponse, error)
}

type SentryMonitoringHandler struct {
	service SentryMonitoringServicer
}

func NewSentryMonitoringHandler(service *service.SentryMonitoringService) *SentryMonitoringHandler {
	return &SentryMonitoringHandler{service: service}
}

func (h *SentryMonitoringHandler) CrashSummary(w http.ResponseWriter, r *http.Request) {
	topN := defaultSentryTopIssueLimit
	if value := r.URL.Query().Get("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil || parsedLimit < 1 || parsedLimit > maxSentryTopIssueLimit {
			respondError(w, http.StatusBadRequest, "INVALID_LIMIT", "limit must be between 1 and 20")
			return
		}
		topN = parsedLimit
	}

	summary, err := h.service.CrashSummary(r.Context(), topN)
	if err != nil {
		respondSentryMonitoringError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, summary)
}

func (h *SentryMonitoringHandler) PerformanceSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.PerformanceSummary(r.Context())
	if err != nil {
		respondSentryMonitoringError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, summary)
}

func respondSentryMonitoringError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrSentryNotConfigured) {
		respondError(w, http.StatusServiceUnavailable, "SENTRY_NOT_CONFIGURED", "Sentry monitoring is not configured")
		return
	}
	respondError(w, http.StatusBadGateway, "SENTRY_UNAVAILABLE", "Failed to load Sentry monitoring data")
}
