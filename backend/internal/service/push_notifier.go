package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
)

const (
	PushEventMessageNew  = "message.new"
	PushEventAdminNotice = "admin.notice"

	pushTemplateMessageNew  = "message_new_default"
	pushTemplateAdminNotice = "admin_notice_default"
	pushTemplateVersion     = 1
	pushDefaultTTLSeconds   = 86400
)

type MessagePushNotifier interface {
	NotifyMessageReceived(messageSeq, recvrSeq, senderSeq int, senderName string, contentPreview string)
}

type PostPushNotifier interface {
	NotifyPostPublished(authorSeq int, postSeq int, subject string)
}

type PushPayload struct {
	EventType       string
	EventID         string
	TemplateKey     string
	TemplateVersion int
	TTLSec          int
	CollapseKey     string
	UserID          string
	Args            map[string]any
	DeepLink        string
	SentAt          time.Time
}

type PushNotification struct {
	DeviceToken     string
	TokenID         int
	Platform        string
	APNsEnvironment string
	BundleID        string
	Title           string
	Body            string
	Payload         PushPayload
}

type MobilePushProvider interface {
	SendPush(ctx context.Context, notification PushNotification) error
}

type mobileDeviceTokenStore interface {
	UpsertToken(usrSeq int, req model.PushDeviceRegistrationRequest) error
	DeactivateToken(usrSeq int, deviceToken string) error
	RevokeToken(deviceToken string) error
	GetActiveTokensByUser(usrSeq int) ([]repository.MobileDeviceToken, error)
	GetActiveTokensForBroadcast(excludeUsrSeq int) ([]repository.MobileDeviceToken, error)
}

type MobilePushService struct {
	tokenRepo mobileDeviceTokenStore
	provider  MobilePushProvider
	logger    zerolog.Logger
	dispatch  func(func())
	now       func() time.Time
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
		dispatch: func(job func()) {
			go job()
		},
		now: time.Now,
	}
}

func NewNoopMobilePushService(tokenRepo *repository.MobileDeviceTokenRepository, logger zerolog.Logger) *MobilePushService {
	return NewMobilePushService(tokenRepo, nil, logger)
}

func (s *MobilePushService) RegisterDeviceToken(usrSeq int, req model.PushDeviceRegistrationRequest) error {
	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	req.DeviceToken = strings.TrimSpace(req.DeviceToken)
	req.APNsEnvironment = normalizeAPNsEnvironment(req.APNsEnvironment)
	req.BundleID = strings.TrimSpace(req.BundleID)
	req.Locale = strings.TrimSpace(req.Locale)

	if req.DeviceToken == "" || req.Platform == "" {
		return fmt.Errorf("device token and platform are required")
	}
	switch req.Platform {
	case "ios":
		if req.APNsEnvironment == "" {
			return fmt.Errorf("apns environment is required")
		}
	case "android":
		req.APNsEnvironment = ""
		req.BundleID = ""
	default:
		return fmt.Errorf("unsupported platform: %s", req.Platform)
	}
	return s.tokenRepo.UpsertToken(usrSeq, req)
}

func (s *MobilePushService) UnregisterDeviceToken(usrSeq int, deviceToken string) error {
	deviceToken = strings.TrimSpace(deviceToken)
	if deviceToken == "" {
		return fmt.Errorf("device token is required")
	}
	return s.tokenRepo.DeactivateToken(usrSeq, deviceToken)
}

func (s *MobilePushService) NotifyMessageReceived(messageSeq, recvrSeq, senderSeq int, senderName string, contentPreview string) {
	s.enqueue(func() {
		s.notifyMessageReceived(context.Background(), messageSeq, recvrSeq, senderSeq, senderName)
	})
}

func (s *MobilePushService) NotifyPostPublished(authorSeq int, postSeq int, subject string) {
	s.enqueue(func() {
		s.notifyPostPublished(context.Background(), authorSeq, postSeq)
	})
}

func (s *MobilePushService) notifyMessageReceived(ctx context.Context, messageSeq, recvrSeq, senderSeq int, senderName string) {
	tokens, err := s.tokenRepo.GetActiveTokensByUser(recvrSeq)
	if err != nil {
		s.logger.Error().Err(err).Int("recvrSeq", recvrSeq).Msg("push: load user tokens failed")
		return
	}

	title := "새 쪽지가 도착했습니다"
	body := "새로운 쪽지가 도착했습니다."
	payload := BuildMessageNewPushPayload(messageSeq, senderSeq, recvrSeq, s.currentTime())

	for _, token := range tokens {
		if token.DeviceToken == "" {
			continue
		}
		notification := pushNotificationForToken(token, title, body, payload)
		if err := s.provider.SendPush(ctx, notification); err != nil {
			s.handleSendError(err, notification)
			continue
		}
		s.logSendResult(notification, "success", "")
	}
}

func (s *MobilePushService) notifyPostPublished(ctx context.Context, authorSeq int, postSeq int) {
	tokens, err := s.tokenRepo.GetActiveTokensForBroadcast(authorSeq)
	if err != nil {
		s.logger.Error().Err(err).Int("authorSeq", authorSeq).Msg("push: load tokens failed for new notice")
		return
	}

	title := "새 공지"
	body := "새 공지가 등록되었습니다."
	sentAt := s.currentTime()
	for _, token := range tokens {
		if token.DeviceToken == "" {
			continue
		}
		payload := BuildAdminNoticePushPayload(postSeq, token.UsrSeq, sentAt)
		notification := pushNotificationForToken(token, title, body, payload)
		if err := s.provider.SendPush(ctx, notification); err != nil {
			s.handleSendError(err, notification)
			continue
		}
		s.logSendResult(notification, "success", "")
	}
}

func (p noopPushProvider) SendPush(ctx context.Context, notification PushNotification) error {
	return nil
}

func (s *MobilePushService) enqueue(job func()) {
	if s.dispatch != nil {
		s.dispatch(job)
		return
	}
	go job()
}

func (s *MobilePushService) currentTime() time.Time {
	if s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func (s *MobilePushService) handleSendError(err error, notification PushNotification) {
	reason := pushErrorReason(err)
	if isInvalidPushTokenError(err) {
		if revokeErr := s.tokenRepo.RevokeToken(notification.DeviceToken); revokeErr != nil {
			s.logger.Error().Err(revokeErr).Int("token_id", notification.TokenID).Str("token_hash", hashPushToken(notification.DeviceToken)).Msg("push: revoke invalid token failed")
		}
	}
	s.logger.Error().
		Err(err).
		Str("event_type", notification.Payload.EventType).
		Str("event_id", notification.Payload.EventID).
		Str("user_id", notification.Payload.UserID).
		Int("token_id", notification.TokenID).
		Str("token_hash", hashPushToken(notification.DeviceToken)).
		Str("platform", notification.Platform).
		Str("push_status", "failure").
		Str("push_reason", reason).
		Msg("push: send failed")
}

func (s *MobilePushService) logSendResult(notification PushNotification, status string, reason string) {
	s.logger.Info().
		Str("event_type", notification.Payload.EventType).
		Str("event_id", notification.Payload.EventID).
		Str("user_id", notification.Payload.UserID).
		Int("token_id", notification.TokenID).
		Str("token_hash", hashPushToken(notification.DeviceToken)).
		Str("platform", notification.Platform).
		Str("push_status", status).
		Str("push_reason", reason).
		Msg("push: send result")
}

func isInvalidPushTokenError(err error) bool {
	var invalidErr interface{ InvalidDeviceToken() bool }
	return errors.As(err, &invalidErr) && invalidErr.InvalidDeviceToken()
}

func pushErrorReason(err error) string {
	var apnsErr *APNsResponseError
	if errors.As(err, &apnsErr) {
		return apnsErr.Reason
	}
	var fcmErr *FCMResponseError
	if errors.As(err, &fcmErr) {
		return fcmErr.Reason
	}
	return ""
}

func BuildMessageNewPushPayload(messageSeq, senderSeq, recvrSeq int, sentAt time.Time) PushPayload {
	return PushPayload{
		EventType:       PushEventMessageNew,
		EventID:         fmt.Sprintf("%s:%d:%d", PushEventMessageNew, messageSeq, recvrSeq),
		TemplateKey:     pushTemplateMessageNew,
		TemplateVersion: pushTemplateVersion,
		TTLSec:          pushDefaultTTLSeconds,
		CollapseKey:     fmt.Sprintf("%s:%d", PushEventMessageNew, recvrSeq),
		UserID:          strconv.Itoa(recvrSeq),
		Args: map[string]any{
			"sender_seq": senderSeq,
			"recvr_seq":  recvrSeq,
		},
		DeepLink: fmt.Sprintf("/messages/%d", senderSeq),
		SentAt:   sentAt.UTC(),
	}
}

func BuildAdminNoticePushPayload(postSeq, recvrSeq int, sentAt time.Time) PushPayload {
	return PushPayload{
		EventType:       PushEventAdminNotice,
		EventID:         fmt.Sprintf("%s:%d", PushEventAdminNotice, postSeq),
		TemplateKey:     pushTemplateAdminNotice,
		TemplateVersion: pushTemplateVersion,
		TTLSec:          pushDefaultTTLSeconds,
		CollapseKey:     fmt.Sprintf("%s:%d", PushEventAdminNotice, postSeq),
		UserID:          strconv.Itoa(recvrSeq),
		Args: map[string]any{
			"post_seq": postSeq,
		},
		DeepLink: fmt.Sprintf("/feed/%d", postSeq),
		SentAt:   sentAt.UTC(),
	}
}

func (p PushPayload) CustomPayload() map[string]any {
	return map[string]any{
		"event_type":       p.EventType,
		"event":            p.EventType,
		"event_id":         p.EventID,
		"template_key":     p.TemplateKey,
		"template_version": p.TemplateVersion,
		"ttl_sec":          p.TTLSec,
		"collapse_key":     p.CollapseKey,
		"user_id":          p.UserID,
		"args":             p.Args,
		"deep_link":        p.DeepLink,
		"sent_at":          p.SentAt.UTC().Format(time.RFC3339),
	}
}

func pushNotificationForToken(token repository.MobileDeviceToken, title string, body string, payload PushPayload) PushNotification {
	return PushNotification{
		DeviceToken:     token.DeviceToken,
		TokenID:         token.MDTSeq,
		Platform:        token.Platform,
		APNsEnvironment: normalizeAPNsEnvironment(token.APNsEnvironment),
		BundleID:        strings.TrimSpace(token.BundleID),
		Title:           title,
		Body:            body,
		Payload:         payload,
	}
}

func normalizeAPNsEnvironment(env string) string {
	env = strings.ToLower(strings.TrimSpace(env))
	switch env {
	case "sandbox", "development", "debug", "dev":
		return "sandbox"
	case "production", "prod", "release", "testflight", "appstore":
		return "production"
	default:
		return ""
	}
}

func hashPushToken(deviceToken string) string {
	sum := sha256.Sum256([]byte(deviceToken))
	return hex.EncodeToString(sum[:])[:16]
}
