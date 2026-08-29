package handler

import (
	"fmt"
	"net/http"

	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type BannerAdHandler struct {
	service *service.BannerAdService
	logger  zerolog.Logger
}

func NewBannerAdHandler(service *service.BannerAdService, logger zerolog.Logger) *BannerAdHandler {
	return &BannerAdHandler{service: service, logger: logger}
}

func (h *BannerAdHandler) GetActive(w http.ResponseWriter, r *http.Request) {
	banner, err := h.service.GetActiveBanner()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get active banner ad")
		return
	}
	if banner == nil {
		respondJSON(w, http.StatusOK, nil)
		return
	}
	respondJSON(w, http.StatusOK, banner)
}

func (h *BannerAdHandler) TrackView(w http.ResponseWriter, r *http.Request) {
	h.trackEvent(w, r, "VIEW")
}

func (h *BannerAdHandler) TrackClick(w http.ResponseWriter, r *http.Request) {
	h.trackEvent(w, r, "CLICK")
}

func (h *BannerAdHandler) trackEvent(w http.ResponseWriter, r *http.Request, eventType string) {
	bnSeq := parseIntParam(chi.URLParam(r, "bnSeq"))
	if bnSeq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid bnSeq")
		return
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.Error().
					Err(fmt.Errorf("%v", recovered)).
					Int("bnSeq", bnSeq).
					Str("eventType", eventType).
					Msg("banner ad event logging panicked")
			}
		}()
		if err := h.service.LogEvent(bnSeq, eventType); err != nil {
			h.logger.Error().
				Err(err).
				Int("bnSeq", bnSeq).
				Str("eventType", eventType).
				Msg("banner ad event logging failed")
		}
	}()
	w.WriteHeader(http.StatusNoContent)
}
