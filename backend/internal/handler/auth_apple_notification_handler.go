package handler

import (
	"encoding/json"
	"net/http"
	"strings"
)

type appleNotificationRequest struct {
	Payload       string `json:"payload"`
	SignedPayload string `json:"signedPayload"`
}

func (r appleNotificationRequest) notificationPayload() string {
	if payload := strings.TrimSpace(r.Payload); payload != "" {
		return payload
	}
	return strings.TrimSpace(r.SignedPayload)
}

func (h *AuthHandler) AppleServerNotification(w http.ResponseWriter, r *http.Request) {
	var request appleNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	signedPayload := request.notificationPayload()
	if signedPayload == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "payload is required")
		return
	}
	notification, err := h.appleVerifier.VerifyServerNotification(r.Context(), signedPayload)
	if err != nil {
		h.logger.Warn().Err(err).Msg("apple server notification verification failed")
		respondError(w, http.StatusUnauthorized, "INVALID_NOTIFICATION", "Invalid Apple notification")
		return
	}
	if err := h.socialLifecycle.ApplyAppleNotification(notification); err != nil {
		h.logger.Warn().Err(err).Str("eventType", notification.Type).Msg("apple server notification handling failed")
		respondError(w, http.StatusBadRequest, "NOTIFICATION_REJECTED", "Apple notification was not applied")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
