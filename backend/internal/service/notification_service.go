// notification_service.go — Business logic for in-app notifications and email enqueue
package service

import (
	"fmt"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
)

// NotificationService orchestrates notification creation, retrieval, and email dispatch.
type NotificationService struct {
	repo        repository.NotificationQuerier
	authRepo    repository.AuthQuerier
	emailQueue  chan<- model.EmailMessage
	logger      zerolog.Logger
	siteBaseURL string
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	repo repository.NotificationQuerier,
	authRepo repository.AuthQuerier,
	emailQueue chan<- model.EmailMessage,
	logger zerolog.Logger,
	siteBaseURL string,
) *NotificationService {
	return &NotificationService{
		repo:        repo,
		authRepo:    authRepo,
		emailQueue:  emailQueue,
		logger:      logger,
		siteBaseURL: siteBaseURL,
	}
}

// NotifyNewMessage creates an in-app notification and enqueues an email
// when a user receives a new direct message.
func (s *NotificationService) NotifyNewMessage(recipientSeq int, senderName string, msgSeq int) {
	title := fmt.Sprintf("%s님이 쪽지를 보냈습니다", senderName)
	refSeq := msgSeq

	if err := s.repo.Insert(recipientSeq, model.NotiTypeNewMessage, title, "", &refSeq); err != nil {
		s.logger.Error().Err(err).Int("recipientSeq", recipientSeq).Msg("failed to insert new message notification")
		return
	}

	s.enqueueEmailForUser(recipientSeq, func(name string) (string, string, error) {
		body, err := RenderNewMessageEmail(NewMessageEmailData{
			RecipientName: name,
			SenderName:    senderName,
			InboxURL:      s.siteBaseURL + "/messages",
		})
		return "새 쪽지가 도착했습니다", body, err
	})
}

// NotifyRegistrationApproved creates an in-app notification and enqueues
// a welcome email when a user's registration is approved.
func (s *NotificationService) NotifyRegistrationApproved(usrSeq int, email string, name string) {
	title := fmt.Sprintf("%s님, 가입이 승인되었습니다", name)

	if err := s.repo.Insert(usrSeq, model.NotiTypeRegistrationApproved, title, "동문 커뮤니티에 오신 것을 환영합니다!", nil); err != nil {
		s.logger.Error().Err(err).Int("usrSeq", usrSeq).Msg("failed to insert registration approved notification")
		return
	}

	if email == "" {
		return
	}
	body, err := RenderApprovalEmail(ApprovalEmailData{
		Name:     name,
		LoginURL: s.siteBaseURL + "/login",
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to render approval email")
		return
	}
	s.trySendEmail(model.EmailMessage{
		To:      email,
		Subject: "가입 승인 완료",
		Body:    body,
	})
}

// NotifyNewComment creates an in-app notification when someone comments
// on a user's post.
func (s *NotificationService) NotifyNewComment(postOwnerSeq int, commenterName string, postSeq int) {
	title := fmt.Sprintf("%s님이 회원님의 게시글에 댓글을 달았습니다", commenterName)
	refSeq := postSeq

	if err := s.repo.Insert(postOwnerSeq, model.NotiTypeNewComment, title, "", &refSeq); err != nil {
		s.logger.Error().Err(err).Int("postOwnerSeq", postOwnerSeq).Msg("failed to insert new comment notification")
	}
}

// NotifyNewLike creates an in-app notification when someone likes
// a user's post.
func (s *NotificationService) NotifyNewLike(postOwnerSeq int, likerName string, postSeq int) {
	title := fmt.Sprintf("%s님이 회원님의 게시글을 좋아합니다", likerName)
	refSeq := postSeq

	if err := s.repo.Insert(postOwnerSeq, model.NotiTypeNewLike, title, "", &refSeq); err != nil {
		s.logger.Error().Err(err).Int("postOwnerSeq", postOwnerSeq).Msg("failed to insert new like notification")
	}
}

// GetNotifications returns a paginated list of notifications for the given user.
func (s *NotificationService) GetNotifications(usrSeq, page, size int) (*model.NotificationListResponse, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 20
	}

	items, totalCount, err := s.repo.GetByUser(usrSeq, page, size)
	if err != nil {
		return nil, err
	}

	unreadCount, err := s.repo.GetUnreadCount(usrSeq)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = []model.Notification{}
	}

	totalPages := totalCount / size
	if totalCount%size > 0 {
		totalPages++
	}

	return &model.NotificationListResponse{
		Items:       items,
		TotalCount:  totalCount,
		UnreadCount: unreadCount,
		Page:        page,
		Size:        size,
		TotalPages:  totalPages,
	}, nil
}

// GetUnreadCount returns the number of unread notifications for the given user.
func (s *NotificationService) GetUnreadCount(usrSeq int) (int, error) {
	return s.repo.GetUnreadCount(usrSeq)
}

// MarkAsRead marks a single notification as read.
func (s *NotificationService) MarkAsRead(anSeq, usrSeq int) error {
	return s.repo.MarkAsRead(anSeq, usrSeq)
}

// MarkAllAsRead marks all unread notifications as read for the given user.
func (s *NotificationService) MarkAllAsRead(usrSeq int) error {
	return s.repo.MarkAllAsRead(usrSeq)
}

// enqueueEmailForUser looks up the user's email and enqueues an email using
// the provided render function to generate subject and body.
func (s *NotificationService) enqueueEmailForUser(usrSeq int, renderFn func(name string) (string, string, error)) {
	user, err := s.authRepo.GetMemberBySeq(usrSeq)
	if err != nil || user == nil {
		s.logger.Warn().Int("usrSeq", usrSeq).Msg("could not look up user for email notification")
		return
	}

	email := ""
	if user.USREmail.Valid {
		email = user.USREmail.String
	}
	if email == "" {
		return
	}

	subject, body, renderErr := renderFn(user.USRName)
	if renderErr != nil {
		s.logger.Error().Err(renderErr).Msg("failed to render email template")
		return
	}
	s.trySendEmail(model.EmailMessage{
		To:      email,
		Subject: subject,
		Body:    body,
	})
}

// trySendEmail performs a non-blocking send to the email queue channel.
func (s *NotificationService) trySendEmail(msg model.EmailMessage) {
	select {
	case s.emailQueue <- msg:
		s.logger.Debug().Str("to", msg.To).Str("subject", msg.Subject).Msg("email enqueued")
	default:
		s.logger.Warn().Str("to", msg.To).Msg("email queue full, dropping email")
	}
}
