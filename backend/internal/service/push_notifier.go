package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type MessagePushNotifier interface {
	NotifyMessageReceived(recvrSeq, senderSeq int, senderName string, contentPreview string)
}

type PostPushNotifier interface {
	NotifyPostPublished(authorSeq int, postSeq int, subject string)
}

type MobilePushProvider interface {
	SendPush(ctx context.Context, deviceToken string, title string, body string, data map[string]any) error
}

type mobileDeviceTokenStore interface {
	UpsertToken(usrSeq int, platform string, deviceToken string, locale string) error
	DeactivateToken(usrSeq int, deviceToken string) error
	RevokeToken(deviceToken string) error
	GetActiveTokensByUser(usrSeq int) ([]repository.MobileDeviceToken, error)
	GetActiveTokensForBroadcast(excludeUsrSeq int) ([]repository.MobileDeviceToken, error)
}

type MobilePushService struct {
	tokenRepo mobileDeviceTokenStore
	provider  MobilePushProvider
	logger    zerolog.Logger
}

type noopPushProvider struct{}

func NewMobilePushService(tokenRepo *repository.MobileDeviceTokenRepository, provider MobilePushProvider, logger zerolog.Logger) *MobilePushService {
	if provider == nil {
		provider = noopPushProvider{}
	}
	return &MobilePushService{
		tokenRepo: tokenRepo,
		provider:  provider,
		logger:    logger,
	}
}

func NewNoopMobilePushService(tokenRepo *repository.MobileDeviceTokenRepository, logger zerolog.Logger) *MobilePushService {
	return NewMobilePushService(tokenRepo, nil, logger)
}

func (s *MobilePushService) RegisterDeviceToken(usrSeq int, req model.PushDeviceRegistrationRequest) error {
	if req.DeviceToken == "" || req.Platform == "" {
		return fmt.Errorf("device token and platform are required")
	}
	if req.Platform != "ios" && req.Platform != "android" {
		return fmt.Errorf("unsupported platform: %s", req.Platform)
	}
	return s.tokenRepo.UpsertToken(usrSeq, req.Platform, req.DeviceToken, req.Locale)
}

func (s *MobilePushService) UnregisterDeviceToken(usrSeq int, deviceToken string) error {
	if deviceToken == "" {
		return fmt.Errorf("device token is required")
	}
	return s.tokenRepo.DeactivateToken(usrSeq, deviceToken)
}

func (s *MobilePushService) NotifyMessageReceived(recvrSeq, senderSeq int, senderName string, contentPreview string) {
	tokens, err := s.tokenRepo.GetActiveTokensByUser(recvrSeq)
	if err != nil {
		s.logger.Error().Err(err).Int("recvrSeq", recvrSeq).Msg("push: load user tokens failed")
		return
	}

	title := "새 메시지"
	body := "새로운 메시지가 도착했습니다."
	data := s.makeMessagePayload(recvrSeq)

	for _, token := range tokens {
		if token.DeviceToken == "" {
			continue
		}
		if err := s.provider.SendPush(context.Background(), token.DeviceToken, title, body, data); err != nil {
			s.handleSendError(err, token.DeviceToken, "message event")
		}
	}
}

func (s *MobilePushService) NotifyPostPublished(authorSeq int, postSeq int, subject string) {
	tokens, err := s.tokenRepo.GetActiveTokensForBroadcast(authorSeq)
	if err != nil {
		s.logger.Error().Err(err).Int("authorSeq", authorSeq).Msg("push: load tokens failed for new post")
		return
	}
	title := "새 공지"
	body := subject
	if body == "" {
		body = "새 게시글이 등록되었습니다."
	}
	for _, token := range tokens {
		if token.DeviceToken == "" {
			continue
		}
		data := s.makePostPayload(token.UsrSeq, authorSeq, postSeq, subject)
		if err := s.provider.SendPush(context.Background(), token.DeviceToken, title, body, data); err != nil {
			s.handleSendError(err, token.DeviceToken, "post event")
		}
	}
}

func (p noopPushProvider) SendPush(ctx context.Context, deviceToken string, title string, body string, data map[string]any) error {
	return nil
}

func (s *MobilePushService) handleSendError(err error, deviceToken string, event string) {
	if isInvalidPushTokenError(err) {
		if revokeErr := s.tokenRepo.RevokeToken(deviceToken); revokeErr != nil {
			s.logger.Error().Err(revokeErr).Str("token", deviceToken).Msg("push: revoke invalid token failed")
		}
	}
	s.logger.Error().Err(err).Str("token", deviceToken).Str("event", event).Msg("push: send failed")
}

func isInvalidPushTokenError(err error) bool {
	var invalidErr interface{ InvalidDeviceToken() bool }
	return errors.As(err, &invalidErr) && invalidErr.InvalidDeviceToken()
}

const pushContractVersion = 1
const pushDefaultTTL = 86400

func (s *MobilePushService) makeMessagePayload(recvrSeq int) map[string]any {
	sentAt := time.Now().Unix()
	eventID := uuid.NewString()

	data := map[string]any{
		"event_type":       "message.new",
		"event":            "message.new",
		"event_id":         eventID,
		"template_key":     "push.message.new",
		"template_version": pushContractVersion,
		"ttl_sec":          pushDefaultTTL,
		"collapse_key":     "message",
		"user_id":          recvrSeq,
		"sent_at":          sentAt,
		"args":             map[string]any{},
		"deep_link":        "/messages",
	}
	return data
}

func (s *MobilePushService) makePostPayload(recvrSeq, authorSeq, postSeq int, subject string) map[string]any {
	sentAt := time.Now().Unix()
	data := map[string]any{
		"event_type":       "admin.notice",
		"event":            "admin.notice",
		"event_id":         uuid.NewString(),
		"template_key":     "push.admin.notice",
		"template_version": pushContractVersion,
		"ttl_sec":          pushDefaultTTL,
		"collapse_key":     "admin.notice",
		"user_id":          recvrSeq,
		"sent_at":          sentAt,
		"args": map[string]any{
			"author_seq": authorSeq,
			"post_seq":   postSeq,
			"subject":    subject,
		},
		"deep_link": "/feed/" + fmt.Sprintf("%d", postSeq),
	}
	return data
}
