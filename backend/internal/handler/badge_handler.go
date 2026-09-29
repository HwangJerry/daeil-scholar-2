// badge_handler.go — HTTP handler for unified badge count aggregation endpoint
package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

type unreadMessageCounter interface {
	GetUnreadCount(usrSeq int) (int, error)
}

type unreadNotificationCounter interface {
	CountUnread(userSeq int) (int, error)
}

// BadgeHandler aggregates unread counts from multiple services into a single response.
type BadgeHandler struct {
	msgService          unreadMessageCounter
	notificationService unreadNotificationCounter
	logger              zerolog.Logger
}

// NewBadgeHandler creates a new BadgeHandler.
func NewBadgeHandler(msgSvc unreadMessageCounter, notificationSvc unreadNotificationCounter, logger zerolog.Logger) *BadgeHandler {
	return &BadgeHandler{msgService: msgSvc, notificationService: notificationSvc, logger: logger}
}

// GetBadges handles GET /api/badges — unified unread counts for polling. A
// failing count degrades to 0 so one source can never hide the other.
func (h *BadgeHandler) GetBadges(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	unreadMessages, err := h.msgService.GetUnreadCount(user.USRSeq)
	if err != nil {
		h.logger.Error().Err(err).Msg("badges: unread messages count failed")
		unreadMessages = 0
	}
	unreadNotifications, err := h.notificationService.CountUnread(user.USRSeq)
	if err != nil {
		h.logger.Error().Err(err).Msg("badges: unread notifications count failed")
		unreadNotifications = 0
	}

	respondJSON(w, http.StatusOK, model.BadgeResponse{
		UnreadMessages:      unreadMessages,
		UnreadNotifications: unreadNotifications,
	})
}
