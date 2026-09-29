// notification_inbox_handler.go — HTTP handlers for the member's notification inbox
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog/log"
)

type NotificationInboxServicer interface {
	List(userSeq, beforeSeq, size int) (*model.NotificationListResponse, error)
	MarkSeenThrough(userSeq, lastSeenPostSeq int) error
}

// markSeenRequest carries the newest postSeq the app has shown. A pointer tells
// a missing field apart from an explicit 0; both are rejected.
type markSeenRequest struct {
	LastSeenPostSeq *int `json:"lastSeenPostSeq"`
}

type NotificationInboxHandler struct {
	service NotificationInboxServicer
}

func NewNotificationInboxHandler(service NotificationInboxServicer) *NotificationInboxHandler {
	return &NotificationInboxHandler{service: service}
}

// List handles GET /api/notifications?cursor=seq_<n>&size=<n>.
func (h *NotificationInboxHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	q := r.URL.Query()
	beforeSeq, ok := parseStrictSeqCursor(q.Get("cursor"))
	if !ok {
		respondError(w, http.StatusBadRequest, "INVALID_CURSOR", "잘못된 페이지입니다.")
		return
	}
	response, err := h.service.List(user.USRSeq, beforeSeq, parseIntParam(q.Get("size")))
	if err != nil {
		log.Error().Err(err).Int("usr_seq", user.USRSeq).Msg("notification list failed")
		respondError(w, http.StatusInternalServerError, "NOTIFICATIONS_FAILED", "알림을 불러오지 못했습니다")
		return
	}
	respondJSON(w, http.StatusOK, response)
}

// MarkSeen handles POST /api/notifications/seen {"lastSeenPostSeq": n} — the
// inbox is seen through the newest notice the app displayed, which clears the
// bell's red dot for it and everything older.
func (h *NotificationInboxHandler) MarkSeen(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	var request markSeenRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.LastSeenPostSeq == nil || *request.LastSeenPostSeq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_LAST_SEEN", "확인한 알림 정보가 올바르지 않습니다")
		return
	}
	if err := h.service.MarkSeenThrough(user.USRSeq, *request.LastSeenPostSeq); err != nil {
		log.Error().Err(err).Int("usr_seq", user.USRSeq).Msg("notification mark seen failed")
		respondError(w, http.StatusInternalServerError, "NOTIFICATIONS_SEEN_FAILED", "알림 확인 처리에 실패했습니다")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
