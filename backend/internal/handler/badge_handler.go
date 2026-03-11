// badge_handler.go — HTTP handler for unified badge count aggregation endpoint
package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

// BadgeHandler aggregates unread counts from multiple services into a single response.
type BadgeHandler struct {
	notifService *service.NotificationService
	msgService   *service.MessageService
	logger       zerolog.Logger
}

// NewBadgeHandler creates a new BadgeHandler.
func NewBadgeHandler(notifSvc *service.NotificationService, msgSvc *service.MessageService, logger zerolog.Logger) *BadgeHandler {
	return &BadgeHandler{notifService: notifSvc, msgService: msgSvc, logger: logger}
}

// GetBadges handles GET /api/badges — unified unread counts for polling.
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

	unreadNotifications, err := h.notifService.GetUnreadCount(user.USRSeq)
	if err != nil {
		h.logger.Error().Err(err).Msg("badges: unread notifications count failed")
		unreadNotifications = 0
	}

	respondJSON(w, http.StatusOK, model.BadgeResponse{
		UnreadMessages:      unreadMessages,
		UnreadNotifications: unreadNotifications,
	})
}
