// admin_subscription_handler.go — Admin endpoint for manually triggering recurring-donation billing
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/job"
	"github.com/rs/zerolog"
)

// AdminSubscriptionHandler exposes manual triggers for the subscription billing job.
// Routes are only registered when ENV=dev so a misuse cannot fire in production.
type AdminSubscriptionHandler struct {
	job    *job.SubscriptionBillingJob
	logger zerolog.Logger
}

// NewAdminSubscriptionHandler creates a new AdminSubscriptionHandler.
func NewAdminSubscriptionHandler(j *job.SubscriptionBillingJob, logger zerolog.Logger) *AdminSubscriptionHandler {
	return &AdminSubscriptionHandler{job: j, logger: logger}
}

// runBillingRequest is the optional JSON body for POST /api/admin/subscription/run-billing.
// `Date` (YYYY-MM-DD) lets dev operators charge a specific BILL_DAY without waiting for the
// real wall clock; omitting it uses the current time.
type runBillingRequest struct {
	Date string `json:"date"`
}

type runBillingResponse struct {
	Date      string   `json:"date"`
	Processed int      `json:"processed"`
	Errors    []string `json:"errors"`
}

// RunBilling handles POST /api/admin/subscription/run-billing.
func (h *AdminSubscriptionHandler) RunBilling(w http.ResponseWriter, r *http.Request) {
	now, err := parseRunBillingDate(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_DATE", err.Error())
		return
	}

	processed, errs := h.job.RunOnce(now)
	errorMsgs := make([]string, 0, len(errs))
	for _, e := range errs {
		errorMsgs = append(errorMsgs, e.Error())
	}
	h.logger.Info().Str("date", now.Format("2006-01-02")).Int("processed", processed).Int("errors", len(errs)).Msg("manual subscription billing run")
	respondJSON(w, http.StatusOK, runBillingResponse{
		Date:      now.Format("2006-01-02"),
		Processed: processed,
		Errors:    errorMsgs,
	})
}

// parseRunBillingDate parses an optional `date` field. Empty body → time.Now().
func parseRunBillingDate(r *http.Request) (time.Time, error) {
	if r.Body == nil || r.ContentLength == 0 {
		return time.Now(), nil
	}
	var req runBillingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return time.Time{}, err
	}
	if req.Date == "" {
		return time.Now(), nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", req.Date, time.Now().Location())
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}
