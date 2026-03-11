// notification_handler.go — HTTP handlers for notification CRUD endpoints
package handler

import (
	"net/http"
	"strconv"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// NotificationHandler handles notification-related HTTP requests.
type NotificationHandler struct {
	service *service.NotificationService
	logger  zerolog.Logger
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(svc *service.NotificationService, logger zerolog.Logger) *NotificationHandler {
	return &NotificationHandler{service: svc, logger: logger}
}

// GetNotifications handles GET /api/notifications.
func (h *NotificationHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	page, size := parsePagination(r)
	result, err := h.service.GetNotifications(user.USRSeq, page, size)
	if err != nil {
		h.logger.Error().Err(err).Msg("get notifications failed")
		respondError(w, http.StatusInternalServerError, "NOTIFICATIONS_FAILED", "알림을 불러올 수 없습니다")
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// GetUnreadCount handles GET /api/notifications/unread-count.
func (h *NotificationHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	count, err := h.service.GetUnreadCount(user.USRSeq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "COUNT_FAILED", "미읽음 개수를 불러올 수 없습니다")
		return
	}

	respondJSON(w, http.StatusOK, map[string]int{"count": count})
}

// MarkAsRead handles PUT /api/notifications/{seq}/read.
func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	seq, err := strconv.Atoi(chi.URLParam(r, "seq"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid notification seq")
		return
	}

	if err := h.service.MarkAsRead(seq, user.USRSeq); err != nil {
		respondError(w, http.StatusInternalServerError, "MARK_READ_FAILED", "읽음 처리에 실패했습니다")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// MarkAllAsRead handles PUT /api/notifications/read-all.
func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	if err := h.service.MarkAllAsRead(user.USRSeq); err != nil {
		respondError(w, http.StatusInternalServerError, "MARK_ALL_READ_FAILED", "전체 읽음 처리에 실패했습니다")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
