// message.go — Alumni direct message domain model
package model

import "time"

// Message represents a row in ALUMNI_MESSAGE table.
type Message struct {
	AMSeq           int    `db:"AM_SEQ" json:"amSeq"`
	ClientMessageID string `db:"AM_CLIENT_MESSAGE_ID" json:"-"`
	SenderSeq       int    `db:"AM_SENDER_SEQ" json:"senderSeq"`
	RecvrSeq        int    `db:"AM_RECVR_SEQ" json:"recvrSeq"`
	Content         string `db:"AM_CONTENT" json:"content"`
	ReadYN          string `db:"AM_READ_YN" json:"readYn"`
	DelSender       string `db:"AM_DEL_SENDER" json:"-"`
	DelRecvr        string `db:"AM_DEL_RECVR" json:"-"`
	RegDate         string `db:"REG_DATE" json:"regDate"`
	ReadDate        string `db:"READ_DATE" json:"readDate"`
	SenderName      string `db:"SENDER_NAME" json:"senderName"`
	RecvrName       string `db:"RECVR_NAME" json:"recvrName"`
}

type MessageParticipant struct {
	UserSeq int    `json:"userSeq"`
	Name    string `json:"name"`
}

type ConversationMessage struct {
	MessageID        int64              `json:"messageId"`
	ClientMessageID  string             `json:"clientMessageId"`
	Sender           MessageParticipant `json:"sender"`
	RecipientUserSeq int                `json:"recipientUserSeq"`
	Content          string             `json:"content"`
	Read             bool               `json:"read"`
	CreatedAt        string             `json:"createdAt"`
	ReadAt           *string            `json:"readAt"`
}

type ConversationMessageListResponse struct {
	Items      []ConversationMessage `json:"items"`
	NextCursor *string               `json:"nextCursor"`
	HasMore    bool                  `json:"hasMore"`
}

type MarkConversationReadRequest struct {
	ThroughMessageID int64 `json:"throughMessageId"`
}

// SendMessageRequest is the request body for POST /api/messages.
type SendMessageRequest struct {
	UserSeq         int    `json:"userSeq"`
	RecvrSeq        int    `json:"recvrSeq,omitempty"`
	ClientMessageID string `json:"clientMessageId"`
	Content         string `json:"content"`
}

func (r SendMessageRequest) RecipientUserSeq() int {
	if r.UserSeq > 0 {
		return r.UserSeq
	}
	return r.RecvrSeq
}

type SendMessageResponse struct {
	MessageID          int64  `db:"AM_SEQ" json:"messageId"`
	ClientMessageID    string `db:"AM_CLIENT_MESSAGE_ID" json:"clientMessageId"`
	Status             string `json:"status"`
	CreatedAt          string `db:"CREATED_AT" json:"createdAt"`
	WasCreated         bool   `db:"-" json:"-"`
	VisibleToRecipient string `db:"AM_VISIBLE_RECVR" json:"-"`
}

// MessageListResponse is the API response for inbox/outbox listing.
type MessageListResponse struct {
	Items      []Message `json:"items"`
	TotalCount int       `json:"totalCount"`
	Page       int       `json:"page"`
	Size       int       `json:"size"`
	TotalPages int       `json:"totalPages"`
}

// ConversationSummary represents a conversation thread summary with another user.
type ConversationSummary struct {
	UserSeq             int       `db:"USER_SEQ" json:"userSeq"`
	Name                string    `db:"NAME" json:"name"`
	LastMessage         string    `db:"LAST_MESSAGE" json:"lastMessage"`
	LastMessageAt       string    `db:"LAST_MESSAGE_AT" json:"lastMessageAt"`
	UnreadCount         int       `db:"UNREAD_COUNT" json:"unreadCount"`
	BlockedByMe         bool      `db:"BLOCKED_BY_ME" json:"blockedByMe"`
	CursorCreatedAt     time.Time `db:"CURSOR_CREATED_AT" json:"-"`
	CursorLastMessageID int64     `db:"CURSOR_MESSAGE_ID" json:"-"`
}

// ConversationListResponse is the API response for GET /api/messages/conversations.
type ConversationListResponse struct {
	Items      []ConversationSummary `json:"items"`
	NextCursor *string               `json:"nextCursor"`
	HasMore    bool                  `json:"hasMore"`
}

// BadgeResponse is the API response for GET /api/badges.
type BadgeResponse struct {
	UnreadMessages int `json:"unreadMessages"`
}
