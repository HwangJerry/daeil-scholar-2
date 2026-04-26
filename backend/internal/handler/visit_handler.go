// visit_handler.go — POST /api/visit/beacon; issues/reads visitor_id cookie and records a hit
package handler

import (
	"net"
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	visitorCookieName = "visitor_id"
	visitorCookieMaxAge = 365 * 24 * 60 * 60
)

type VisitHandler struct {
	service *service.VisitService
	logger  zerolog.Logger
	secure  bool
}

func NewVisitHandler(svc *service.VisitService, logger zerolog.Logger, secure bool) *VisitHandler {
	return &VisitHandler{service: svc, logger: logger, secure: secure}
}

// Beacon records one page visit. Anonymous visitors get a fresh visitor_id
// cookie on first call; logged-in users are additionally tagged with USR_SEQ
// inside the same-day visit row.
func (h *VisitHandler) Beacon(w http.ResponseWriter, r *http.Request) {
	visitorID := readVisitorID(r)
	if visitorID == "" {
		visitorID = uuid.New().String()
		http.SetCookie(w, &http.Cookie{
			Name:     visitorCookieName,
			Value:    visitorID,
			Path:     "/",
			HttpOnly: true,
			Secure:   h.secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   visitorCookieMaxAge,
			Expires:  time.Now().Add(time.Duration(visitorCookieMaxAge) * time.Second),
		})
	}

	usrSeq := 0
	if user := middleware.GetAuthUser(r.Context()); user != nil {
		usrSeq = user.USRSeq
	}

	ipAddr := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ipAddr = host
	}
	userAgent := r.Header.Get("User-Agent")

	if err := h.service.RecordVisit(visitorID, usrSeq, userAgent, ipAddr); err != nil {
		h.logger.Error().Err(err).Msg("visit beacon: record failed")
	}

	w.WriteHeader(http.StatusNoContent)
}

func readVisitorID(r *http.Request) string {
	c, err := r.Cookie(visitorCookieName)
	if err != nil || c == nil {
		return ""
	}
	if _, err := uuid.Parse(c.Value); err != nil {
		return ""
	}
	return c.Value
}
