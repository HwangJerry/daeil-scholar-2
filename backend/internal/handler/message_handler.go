// message_handler.go — HTTP handlers for alumni direct messaging
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-chi/chi/v5"
)

// MessageServicer defines the messaging operations used by MessageHandler.
// Defined here (consumer side) so tests can inject a stub without importing
// the service package.
type MessageServicer interface {
	SendMessage(senderSeq int, senderName string, req model.SendMessageRequest) error
	GetInbox(usrSeq, page, size int) (*model.MessageListResponse, error)
	GetOutbox(usrSeq, page, size int) (*model.MessageListResponse, error)
	MarkAsRead(amSeq, usrSeq int) error
	DeleteMessage(amSeq, usrSeq int) error
	GetConversations(usrSeq int) (*model.ConversationListResponse, error)
	GetConversationMessages(usrSeq, otherSeq, page, size int) (*model.MessageListResponse, error)
	MarkConversationRead(usrSeq, senderSeq int) error
}

// MessageHandler handles message-related HTTP requests.
type MessageHandler struct {
	service MessageServicer
}

// NewMessageHandler creates a new MessageHandler.
func NewMessageHandler(svc MessageServicer) *MessageHandler {
	return &MessageHandler{service: svc}
}

// Send handles POST /api/messages.
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	var req model.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.SendMessage(user.USRSeq, user.USRName, req); err != nil {
		var ve *model.ValidationError
		if errors.As(err, &ve) {
			respondError(w, http.StatusBadRequest, "SEND_FAILED", ve.Error())
		} else {
			respondError(w, http.StatusInternalServerError, "SEND_FAILED", "메시지 전송에 실패했습니다")
		}
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetInbox handles GET /api/messages/inbox.
func (h *MessageHandler) GetInbox(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	page, size := parsePagination(r)
	result, err := h.service.GetInbox(user.USRSeq, page, size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INBOX_FAILED", "Failed to load inbox")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// GetOutbox handles GET /api/messages/outbox.
func (h *MessageHandler) GetOutbox(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	page, size := parsePagination(r)
	result, err := h.service.GetOutbox(user.USRSeq, page, size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "OUTBOX_FAILED", "Failed to load outbox")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// MarkAsRead handles PUT /api/messages/{seq}/read.
func (h *MessageHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	seq, err := strconv.Atoi(chi.URLParam(r, "seq"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid message seq")
		return
	}
	if err := h.service.MarkAsRead(seq, user.USRSeq); err != nil {
		respondError(w, http.StatusInternalServerError, "MARK_READ_FAILED", "Failed to mark as read")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Delete handles DELETE /api/messages/{seq}.
func (h *MessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	seq, err := strconv.Atoi(chi.URLParam(r, "seq"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid message seq")
		return
	}
	if err := h.service.DeleteMessage(seq, user.USRSeq); err != nil {
		respondError(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete message")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetConversations handles GET /api/messages/conversations.
func (h *MessageHandler) GetConversations(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	result, err := h.service.GetConversations(user.USRSeq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "CONVERSATIONS_FAILED", "Failed to load conversations")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// GetConversationMessages handles GET /api/messages/conversations/{userSeq}.
func (h *MessageHandler) GetConversationMessages(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	otherSeq, err := strconv.Atoi(chi.URLParam(r, "userSeq"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid user seq")
		return
	}
	page, size := parsePagination(r)
	result, err := h.service.GetConversationMessages(user.USRSeq, otherSeq, page, size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "CONVERSATION_MESSAGES_FAILED", "Failed to load conversation messages")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// MarkConversationRead handles PUT /api/messages/conversations/{userSeq}/read.
func (h *MessageHandler) MarkConversationRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	senderSeq, err := strconv.Atoi(chi.URLParam(r, "userSeq"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid user seq")
		return
	}
	if err := h.service.MarkConversationRead(user.USRSeq, senderSeq); err != nil {
		respondError(w, http.StatusInternalServerError, "MARK_READ_FAILED", "Failed to mark conversation as read")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func parsePagination(r *http.Request) (int, int) {
	page := 1
	size := 20
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && s > 0 {
		size = s
	}
	return page, size
}
