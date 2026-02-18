package handler

import (
	"net"
	"net/http"
	"strconv"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdHandler struct {
	service *service.AdService
}

func NewAdHandler(service *service.AdService) *AdHandler {
	return &AdHandler{service: service}
}

func (h *AdHandler) TrackView(w http.ResponseWriter, r *http.Request) {
	h.trackEvent(w, r, "VIEW")
}

func (h *AdHandler) TrackClick(w http.ResponseWriter, r *http.Request) {
	h.trackEvent(w, r, "CLICK")
}

func (h *AdHandler) trackEvent(w http.ResponseWriter, r *http.Request, eventType string) {
	maSeq, err := strconv.Atoi(chi.URLParam(r, "maSeq"))
	if err != nil || maSeq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid maSeq")
		return
	}
	usrSeq := 0
	if user := middleware.GetAuthUser(r.Context()); user != nil {
		usrSeq = user.USRSeq
	}
	ipAddr := r.RemoteAddr
	if host, _, splitErr := net.SplitHostPort(r.RemoteAddr); splitErr == nil {
		ipAddr = host
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// silently recover — ad tracking should not crash the server
			}
		}()
		h.service.LogEvent(maSeq, usrSeq, eventType, ipAddr)
	}()
	w.WriteHeader(http.StatusNoContent)
}
