// mobile_app_event_handler.go — Mobile event ingestion and admin aggregation.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

const mobileEventDateLayout = "2006-01-02"

type MobileAppEventServicer interface {
	RecordBatch(events []model.MobileAppEvent, authenticatedUserID *int) error
	Summary(from, to time.Time, platform, eventType string) ([]model.MobileAppEventSummary, error)
}

type MobileAppEventHandler struct {
	service MobileAppEventServicer
}

func NewMobileAppEventHandler(service *service.MobileAppEventService) *MobileAppEventHandler {
	return &MobileAppEventHandler{service: service}
}

func (h *MobileAppEventHandler) Collect(w http.ResponseWriter, r *http.Request) {
	var request model.MobileAppEventBatchRequest
	if err := decodeClosedMobileEventJSON(r, &request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	var authenticatedUserID *int
	if user := middleware.GetAuthUser(r.Context()); user != nil {
		userID := user.USRSeq
		authenticatedUserID = &userID
	}
	if err := h.service.RecordBatch(request.Events, authenticatedUserID); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidMobileEventBatch):
			respondError(w, http.StatusBadRequest, "INVALID_BATCH", "events must contain between 1 and 100 items")
		case errors.Is(err, service.ErrInvalidMobileEvent):
			respondError(w, http.StatusBadRequest, "INVALID_EVENT", "One or more events are invalid")
		default:
			respondError(w, http.StatusInternalServerError, "EVENT_STORE_FAILED", "Failed to store mobile events")
		}
		return
	}

	respondJSON(w, http.StatusAccepted, model.MobileAppEventBatchResponse{Accepted: len(request.Events)})
}

// Summary accepts an inclusive ?from=YYYY-MM-DD&to=YYYY-MM-DD range and
// defaults to the trailing 30 KST calendar days.
func (h *MobileAppEventHandler) Summary(w http.ResponseWriter, r *http.Request) {
	today := service.Today()
	from := today.AddDate(0, 0, -29)
	to := today

	var err error
	if value := r.URL.Query().Get("from"); value != "" {
		from, err = time.ParseInLocation(mobileEventDateLayout, value, today.Location())
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_DATE", "from must use YYYY-MM-DD")
			return
		}
	}
	if value := r.URL.Query().Get("to"); value != "" {
		to, err = time.ParseInLocation(mobileEventDateLayout, value, today.Location())
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_DATE", "to must use YYYY-MM-DD")
			return
		}
	}
	if to.Before(from) {
		respondError(w, http.StatusBadRequest, "INVALID_RANGE", "to must be on or after from")
		return
	}

	platform := r.URL.Query().Get("platform")
	eventType := r.URL.Query().Get("event_type")
	if camelEventType := r.URL.Query().Get("eventType"); camelEventType != "" {
		if eventType != "" && eventType != camelEventType {
			respondError(w, http.StatusBadRequest, "INVALID_FILTER", "event_type and eventType must match")
			return
		}
		eventType = camelEventType
	}

	items, err := h.service.Summary(from, to, platform, eventType)
	if err != nil {
		if errors.Is(err, service.ErrInvalidMobileEventFilter) {
			respondError(w, http.StatusBadRequest, "INVALID_FILTER", "platform or event_type is invalid")
			return
		}
		respondError(w, http.StatusInternalServerError, "SUMMARY_FAILED", "Failed to load mobile event summary")
		return
	}
	respondJSON(w, http.StatusOK, model.MobileAppEventSummaryResponse{
		From:      from.Format(mobileEventDateLayout),
		To:        to.Format(mobileEventDateLayout),
		Platform:  platform,
		EventType: eventType,
		Items:     items,
	})
}

func decodeClosedMobileEventJSON(r *http.Request, target interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one object")
	}
	return nil
}
