// message_service.go — Business logic for alumni direct messaging
package service

import (
	"errors"

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

// SendMessage validates and sends a message, then triggers a notification.
func (s *MessageService) SendMessage(senderSeq int, senderName string, req model.SendMessageRequest) error {
	if req.Content == "" {
		return errors.New("메시지 내용을 입력해주세요")
	}
	if len([]rune(req.Content)) > maxMessageLength {
		return errors.New("메시지는 1000자 이하로 입력해주세요")
	}
	if req.RecvrSeq <= 0 {
		return errors.New("수신자를 지정해주세요")
	}
	if senderSeq == req.RecvrSeq {
		return errors.New("자기 자신에게는 쪽지를 보낼 수 없습니다")
	}

	exists, err := s.profileRepo.CheckUserExists(req.RecvrSeq)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("존재하지 않는 회원입니다")
	}

	if err := s.repo.InsertMessage(senderSeq, req.RecvrSeq, req.Content); err != nil {
		return err
	}

	s.notifier.NotifyMessageReceived(req.RecvrSeq, senderSeq, senderName)
	s.notifier.NotifyMessageSent(senderSeq, req.RecvrSeq)

	return nil
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
	return s.repo.MarkAsRead(amSeq, usrSeq)
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
func (s *MessageService) GetConversations(usrSeq int) (*model.ConversationListResponse, error) {
	conversations, err := s.repo.GetConversations(usrSeq)
	if err != nil {
		return nil, err
	}
	if conversations == nil {
		conversations = []model.ConversationSummary{}
	}
	return &model.ConversationListResponse{Items: conversations}, nil
}

// GetConversationMessages returns paginated messages in a conversation.
func (s *MessageService) GetConversationMessages(usrSeq, otherSeq, page, size int) (*model.MessageListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 30
	}
	if size > 50 {
		size = 50
	}
	messages, total, err := s.repo.GetConversationMessages(usrSeq, otherSeq, page, size)
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

// MarkConversationRead marks all messages from senderSeq to usrSeq as read.
func (s *MessageService) MarkConversationRead(usrSeq, senderSeq int) error {
	return s.repo.MarkConversationRead(usrSeq, senderSeq)
}
