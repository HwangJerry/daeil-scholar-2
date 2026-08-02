// realtime_handler.go — SSE endpoint streaming per-user message events to the client
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/realtime"
	"github.com/rs/zerolog"
)

const sseKeepaliveInterval = 25 * time.Second

type RealtimeHandler struct {
	hub    *realtime.Hub
	logger zerolog.Logger
}

func NewRealtimeHandler(hub *realtime.Hub, logger zerolog.Logger) *RealtimeHandler {
	return &RealtimeHandler{hub: hub, logger: logger}
}

// Stream handles GET /api/messages/stream — opens an SSE connection that pushes
// message lifecycle events whenever the authenticated user sends, receives, or
// has their sent messages read.
func (h *RealtimeHandler) Stream(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthUserOrError(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	// Disable the server's WriteTimeout (60s in main.go) for this long-lived response.
	if rc := http.NewResponseController(w); rc != nil {
		_ = rc.SetWriteDeadline(time.Time{})
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	lastEventID, err := strconv.ParseInt(strings.TrimSpace(r.Header.Get("Last-Event-ID")), 10, 64)
	var sub *realtime.Subscriber
	if err == nil && lastEventID > 0 {
		sub = h.hub.Subscribe(user.USRSeq, lastEventID)
	} else {
		sub = h.hub.Subscribe(user.USRSeq)
	}
	defer h.hub.Unsubscribe(sub)

	if _, err := fmt.Fprint(w, "event: ready\ndata: {\"ok\":true}\n\n"); err != nil {
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(sseKeepaliveInterval)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case ev, ok := <-sub.Ch:
			if !ok {
				return
			}
			payload, ok := ev.Payload.(map[string]any)
			if !ok {
				h.logger.Warn().Str("eventType", ev.Type).Msg("realtime payload missing canonical event ID")
				continue
			}
			eventID, ok := payload["eventId"].(int64)
			if !ok || eventID <= 0 {
				h.logger.Warn().Str("eventType", ev.Type).Msg("realtime payload has invalid canonical event ID")
				continue
			}
			data, err := json.Marshal(ev.Payload)
			if err != nil {
				h.logger.Warn().Err(err).Str("eventType", ev.Type).Msg("realtime payload marshal failed")
				continue
			}
			if _, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", eventID, ev.Type, data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
