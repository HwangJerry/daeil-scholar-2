// message_service.go — Business logic for alumni direct messaging
package service

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

const maxMessageLength = 1000

// MessageService handles direct messaging business logic.
type MessageService struct {
	repo        repository.MessageQuerier
	profileRepo repository.ProfileQuerier
	notifier    MessageNotifier
}

// NewMessageService creates a new MessageService.
func NewMessageService(repo repository.MessageQuerier, profileRepo repository.ProfileQuerier, notifier MessageNotifier) *MessageService {
	if notifier == nil {
		notifier = nopMessageNotifier{}
	}
	return &MessageService{repo: repo, profileRepo: profileRepo, notifier: notifier}
}

// SendMessage validates and accepts a message idempotently, then triggers a
// notification only for the first acceptance.
func (s *MessageService) SendMessage(senderSeq int, senderName string, req model.SendMessageRequest) (*model.SendMessageResponse, error) {
	if req.Content == "" {
		return nil, &model.ValidationError{Msg: "메시지 내용을 입력해주세요"}
	}
	if len([]rune(req.Content)) > maxMessageLength {
		return nil, &model.ValidationError{Msg: "메시지는 1000자 이하로 입력해주세요"}
	}
	if req.ClientMessageID == "" || len(req.ClientMessageID) > 64 {
		return nil, &model.ValidationError{Msg: "clientMessageId가 올바르지 않습니다"}
	}
	existing, err := s.repo.FindAcceptedMessage(senderSeq, req.ClientMessageID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	recipientSeq := req.RecipientUserSeq()
	if recipientSeq <= 0 {
		return nil, &model.ValidationError{Msg: "수신자를 지정해주세요"}
	}
	if senderSeq == recipientSeq {
		return nil, &model.ValidationError{Msg: "자기 자신에게는 쪽지를 보낼 수 없습니다"}
	}

	exists, err := s.repo.IsApprovedAlumni(recipientSeq)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &model.ValidationError{Msg: "승인된 동문이 아닙니다"}
	}

	accepted, err := s.repo.AcceptMessage(senderSeq, recipientSeq, req.ClientMessageID, req.Content)
	if err != nil {
		return nil, err
	}

	if accepted.WasCreated && accepted.VisibleToRecipient == "Y" {
		s.notifier.NotifyMessageReceived(recipientSeq, senderSeq, senderName, accepted, req.Content)
		s.notifier.NotifyMessageSent(senderSeq, recipientSeq)
	}

	return accepted, nil
}

// GetInbox returns paginated inbox messages.
func (s *MessageService) GetInbox(usrSeq int, page int, size int) (*model.MessageListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	messages, total, err := s.repo.GetInbox(usrSeq, page, size)
	if err != nil {
		return nil, err
	}
	if messages == nil {
		messages = []model.Message{}
	}
	totalPages := 0
	if size > 0 {
		totalPages = (total + size - 1) / size
	}
	return &model.MessageListResponse{
		Items:      messages,
		TotalCount: total,
		Page:       page,
		Size:       size,
		TotalPages: totalPages,
	}, nil
}

// GetOutbox returns paginated outbox messages.
func (s *MessageService) GetOutbox(usrSeq int, page int, size int) (*model.MessageListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	messages, total, err := s.repo.GetOutbox(usrSeq, page, size)
	if err != nil {
		return nil, err
	}
	if messages == nil {
		messages = []model.Message{}
	}
	totalPages := 0
	if size > 0 {
		totalPages = (total + size - 1) / size
	}
	return &model.MessageListResponse{
		Items:      messages,
		TotalCount: total,
		Page:       page,
		Size:       size,
		TotalPages: totalPages,
	}, nil
}

// MarkAsRead marks a message as read.
func (s *MessageService) MarkAsRead(amSeq int, usrSeq int) error {
	senderSeq, changed, err := s.repo.MarkAsRead(amSeq, usrSeq)
	if err != nil {
		return err
	}
	if changed {
		s.notifier.NotifyMessagesRead(senderSeq, usrSeq, int64(amSeq), time.Now().UTC().Format(time.RFC3339))
	}
	return nil
}

// DeleteMessage soft-deletes a message for the requesting user.
func (s *MessageService) DeleteMessage(amSeq int, usrSeq int) error {
	return s.repo.DeleteMessage(amSeq, usrSeq)
}

// GetUnreadCount returns the number of unread messages.
func (s *MessageService) GetUnreadCount(usrSeq int) (int, error) {
	return s.repo.GetUnreadCount(usrSeq)
}

// GetConversations returns conversation summaries for the authenticated user.
func (s *MessageService) GetConversations(usrSeq int, cursor string, size int) (*model.ConversationListResponse, error) {
	var cursorPoint *messageCursor
	if cursor != "" {
		decoded, err := decodeMessageCursor(cursor)
		if err != nil {
			return nil, &model.ValidationError{Msg: "cursor가 올바르지 않습니다"}
		}
		cursorPoint = decoded
	}
	if size <= 0 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	var beforeCreatedAt *time.Time
	var beforeMessageID int64
	if cursorPoint != nil {
		beforeCreatedAt = &cursorPoint.CreatedAt
		beforeMessageID = cursorPoint.MessageID
	}
	conversations, err := s.repo.GetConversations(usrSeq, beforeCreatedAt, beforeMessageID, size+1)
	if err != nil {
		return nil, err
	}
	if conversations == nil {
		conversations = []model.ConversationSummary{}
	}
	if cursorPoint != nil {
		filtered := make([]model.ConversationSummary, 0, len(conversations))
		for _, conversation := range conversations {
			createdAt := conversation.CursorCreatedAt.UTC()
			if createdAt.Before(cursorPoint.CreatedAt) ||
				(createdAt.Equal(cursorPoint.CreatedAt) && conversation.CursorLastMessageID < cursorPoint.MessageID) {
				filtered = append(filtered, conversation)
			}
		}
		conversations = filtered
	}
	response := &model.ConversationListResponse{Items: conversations}
	if len(conversations) > size {
		response.HasMore = true
		response.Items = conversations[:size]
		last := response.Items[len(response.Items)-1]
		nextCursor, err := encodeMessageCursor(last.CursorCreatedAt, last.CursorLastMessageID)
		if err != nil {
			return nil, err
		}
		response.NextCursor = &nextCursor
	}
	return response, nil
}

type messageCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	MessageID int64     `json:"messageId"`
}

func encodeMessageCursor(createdAt time.Time, messageID int64) (string, error) {
	payload, err := json.Marshal(messageCursor{CreatedAt: createdAt.UTC(), MessageID: messageID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeMessageCursor(encoded string) (*messageCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var cursor messageCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, err
	}
	if cursor.CreatedAt.IsZero() || cursor.MessageID <= 0 {
		return nil, &model.ValidationError{Msg: "cursor가 올바르지 않습니다"}
	}
	cursor.CreatedAt = cursor.CreatedAt.UTC()
	return &cursor, nil
}

// GetConversationMessages returns canonical messages in a conversation.
func (s *MessageService) GetConversationMessages(usrSeq, otherSeq int, before string, size int) (*model.ConversationMessageListResponse, error) {
	var cursorPoint *messageCursor
	if before != "" {
		decoded, err := decodeMessageCursor(before)
		if err != nil {
			return nil, &model.ValidationError{Msg: "before cursor가 올바르지 않습니다"}
		}
		cursorPoint = decoded
	}
	if size <= 0 {
		size = 30
	}
	if size > 50 {
		size = 50
	}
	var beforeCreatedAt *time.Time
	var beforeMessageID int64
	if cursorPoint != nil {
		beforeCreatedAt = &cursorPoint.CreatedAt
		beforeMessageID = cursorPoint.MessageID
	}
	messages, err := s.repo.GetCanonicalConversationMessages(usrSeq, otherSeq, beforeCreatedAt, beforeMessageID, size+1)
	if err != nil {
		return nil, err
	}
	if cursorPoint != nil {
		filtered := make([]model.Message, 0, len(messages))
		for _, message := range messages {
			createdAt, err := time.Parse(time.RFC3339, message.RegDate)
			if err != nil {
				return nil, err
			}
			createdAt = createdAt.UTC()
			if createdAt.Before(cursorPoint.CreatedAt) ||
				(createdAt.Equal(cursorPoint.CreatedAt) && int64(message.AMSeq) < cursorPoint.MessageID) {
				filtered = append(filtered, message)
			}
		}
		messages = filtered
	}
	hasMore := len(messages) > size
	if hasMore {
		messages = messages[:size]
	}
	items := make([]model.ConversationMessage, 0, len(messages))
	for _, message := range messages {
		var readAt *string
		if message.ReadDate != "" {
			value := message.ReadDate
			readAt = &value
		}
		items = append(items, model.ConversationMessage{
			MessageID:        int64(message.AMSeq),
			ClientMessageID:  message.ClientMessageID,
			Sender:           model.MessageParticipant{UserSeq: message.SenderSeq, Name: message.SenderName},
			RecipientUserSeq: message.RecvrSeq,
			Content:          message.Content,
			Read:             message.ReadYN == "Y",
			CreatedAt:        message.RegDate,
			ReadAt:           readAt,
		})
	}
	response := &model.ConversationMessageListResponse{Items: items, HasMore: hasMore}
	if hasMore {
		last := messages[len(messages)-1]
		createdAt, err := time.Parse(time.RFC3339, last.RegDate)
		if err != nil {
			return nil, err
		}
		nextCursor, err := encodeMessageCursor(createdAt, int64(last.AMSeq))
		if err != nil {
			return nil, err
		}
		response.NextCursor = &nextCursor
	}
	return response, nil
}

// MarkConversationRead marks all messages from senderSeq to usrSeq as read.
func (s *MessageService) MarkConversationRead(usrSeq, senderSeq int, throughMessageID int64) error {
	changed, err := s.repo.MarkConversationRead(usrSeq, senderSeq, throughMessageID)
	if err != nil {
		return err
	}
	if changed {
		s.notifier.NotifyMessagesRead(senderSeq, usrSeq, throughMessageID, time.Now().UTC().Format(time.RFC3339))
	}
	return nil
}
