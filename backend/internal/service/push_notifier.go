package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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

	pushTemplateMessageNew  = "push.message.new"
	pushTemplateAdminNotice = "push.admin.notice"
	pushTemplateVersion     = 1
	pushDefaultTTLSeconds   = 86400
)

var ErrInvalidPushPreferences = errors.New("invalid push preferences")

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

type pushOutboxStore interface {
	Enqueue(ctx context.Context, job repository.PushOutboxInsert) error
}

type pushPreferenceStore interface {
	GetPreferences(usrSeq int) (model.PushPreferences, error)
	UpsertPreferences(usrSeq int, preferences model.PushPreferences) (model.PushPreferences, error)
}

type mobileDeviceTokenStore interface {
	UpsertToken(usrSeq int, req model.PushDeviceRegistrationRequest) error
	DeactivateToken(usrSeq int, deviceToken string) error
	RevokeToken(deviceToken string) error
	GetActiveTokensByUser(usrSeq int) ([]repository.MobileDeviceToken, error)
	GetActiveTokensForBroadcast(excludeUsrSeq int) ([]repository.MobileDeviceToken, error)
}

type MobilePushService struct {
	tokenRepo   mobileDeviceTokenStore
	preferences pushPreferenceStore
	outbox      pushOutboxStore
	provider    MobilePushProvider
	logger      zerolog.Logger
	dispatch    func(func())
	now         func() time.Time
}

type noopPushProvider struct{}

func NewMobilePushService(tokenRepo *repository.MobileDeviceTokenRepository, provider MobilePushProvider, logger zerolog.Logger) *MobilePushService {
	return NewMobilePushServiceWithOutbox(tokenRepo, nil, provider, logger)
}

func NewMobilePushServiceWithOutbox(tokenRepo *repository.MobileDeviceTokenRepository, outbox *repository.PushOutboxRepository, provider MobilePushProvider, logger zerolog.Logger) *MobilePushService {
	return NewMobilePushServiceWithOutboxAndPreferences(tokenRepo, nil, outbox, provider, logger)
}

func NewMobilePushServiceWithOutboxAndPreferences(
	tokenRepo *repository.MobileDeviceTokenRepository,
	preferences *repository.PushPreferenceRepository,
	outbox *repository.PushOutboxRepository,
	provider MobilePushProvider,
	logger zerolog.Logger,
) *MobilePushService {
	if provider == nil {
		provider = noopPushProvider{}
	}
	return &MobilePushService{
		tokenRepo:   tokenRepo,
		preferences: preferences,
		outbox:      outbox,
		provider:    provider,
		logger:      logger,
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

func (s *MobilePushService) GetPreferences(usrSeq int) (model.PushPreferences, error) {
	if s.preferences == nil {
		return model.DefaultPushPreferences(), nil
	}
	return s.preferences.GetPreferences(usrSeq)
}

func (s *MobilePushService) UpdatePreferences(usrSeq int, req model.PushPreferencesUpdateRequest) (model.PushPreferences, error) {
	preferences, ok := req.Preferences()
	if !ok {
		return model.PushPreferences{}, ErrInvalidPushPreferences
	}
	if s.preferences == nil {
		return preferences, nil
	}
	return s.preferences.UpsertPreferences(usrSeq, preferences)
}

func (s *MobilePushService) NotifyMessageReceived(messageSeq, recvrSeq, senderSeq int, senderName string, contentPreview string) {
	if s.outbox != nil {
		s.notifyMessageReceived(context.Background(), messageSeq, recvrSeq, senderSeq, senderName)
		return
	}
	s.enqueue(func() {
		s.notifyMessageReceived(context.Background(), messageSeq, recvrSeq, senderSeq, senderName)
	})
}

func (s *MobilePushService) NotifyPostPublished(authorSeq int, postSeq int, subject string) {
	if s.outbox != nil {
		s.notifyPostPublished(context.Background(), authorSeq, postSeq)
		return
	}
	s.enqueue(func() {
		s.notifyPostPublished(context.Background(), authorSeq, postSeq)
	})
}

func (s *MobilePushService) notifyMessageReceived(ctx context.Context, messageSeq, recvrSeq, senderSeq int, senderName string) {
	if !s.isMessagePushEnabled(recvrSeq) {
		s.logger.Info().Int("user_id", recvrSeq).Str("event_type", PushEventMessageNew).Msg("push: skipped by user preference")
		return
	}

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
		if s.outbox != nil && isIOSPushToken(token) {
			s.enqueueOutboxNotification(ctx, notification)
			continue
		}
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
	noticePreferenceCache := map[int]bool{}
	for _, token := range tokens {
		if token.DeviceToken == "" {
			continue
		}
		if !s.isNoticePushEnabled(token.UsrSeq, noticePreferenceCache) {
			s.logger.Info().Int("user_id", token.UsrSeq).Str("event_type", PushEventAdminNotice).Msg("push: skipped by user preference")
			continue
		}
		payload := BuildAdminNoticePushPayload(postSeq, token.UsrSeq, sentAt)
		notification := pushNotificationForToken(token, title, body, payload)
		if s.outbox != nil && isIOSPushToken(token) {
			s.enqueueOutboxNotification(ctx, notification)
			continue
		}
		if err := s.provider.SendPush(ctx, notification); err != nil {
			s.handleSendError(err, notification)
			continue
		}
		s.logSendResult(notification, "success", "")
	}
}

func (s *MobilePushService) isMessagePushEnabled(usrSeq int) bool {
	preferences, ok := s.preferencesForUser(usrSeq)
	return ok && preferences.MessageEnabled
}

func (s *MobilePushService) isNoticePushEnabled(usrSeq int, cache map[int]bool) bool {
	if enabled, ok := cache[usrSeq]; ok {
		return enabled
	}
	preferences, ok := s.preferencesForUser(usrSeq)
	enabled := ok && preferences.NoticeEnabled
	cache[usrSeq] = enabled
	return enabled
}

func (s *MobilePushService) preferencesForUser(usrSeq int) (model.PushPreferences, bool) {
	preferences, err := s.GetPreferences(usrSeq)
	if err != nil {
		s.logger.Error().Err(err).Int("user_id", usrSeq).Msg("push: load user preferences failed")
		return model.PushPreferences{}, false
	}
	return preferences, true
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
	reason := PushErrorReason(err)
	if IsInvalidDeviceToken(err) {
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

func (s *MobilePushService) enqueueOutboxNotification(ctx context.Context, notification PushNotification) {
	payloadJSON, err := marshalPushPayload(notification.Payload)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("event_type", notification.Payload.EventType).
			Str("event_id", notification.Payload.EventID).
			Str("user_id", notification.Payload.UserID).
			Int("token_id", notification.TokenID).
			Str("token_hash", hashPushToken(notification.DeviceToken)).
			Msg("push: payload marshal failed")
		return
	}
	err = s.outbox.Enqueue(ctx, repository.PushOutboxInsert{
		EventType:       notification.Payload.EventType,
		EventID:         notification.Payload.EventID,
		UsrSeq:          parsePushUserID(notification.Payload.UserID),
		MDTSeq:          notification.TokenID,
		DeviceToken:     notification.DeviceToken,
		APNsEnvironment: notification.APNsEnvironment,
		BundleID:        notification.BundleID,
		Title:           notification.Title,
		Body:            notification.Body,
		PayloadJSON:     payloadJSON,
	})
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("event_type", notification.Payload.EventType).
			Str("event_id", notification.Payload.EventID).
			Str("user_id", notification.Payload.UserID).
			Int("token_id", notification.TokenID).
			Str("token_hash", hashPushToken(notification.DeviceToken)).
			Msg("push: outbox enqueue failed")
		return
	}
	s.logger.Info().
		Str("event_type", notification.Payload.EventType).
		Str("event_id", notification.Payload.EventID).
		Str("user_id", notification.Payload.UserID).
		Int("token_id", notification.TokenID).
		Str("token_hash", hashPushToken(notification.DeviceToken)).
		Str("push_status", "PENDING").
		Msg("push: outbox enqueued")
}

func IsInvalidDeviceToken(err error) bool {
	var invalidErr interface{ InvalidDeviceToken() bool }
	return errors.As(err, &invalidErr) && invalidErr.InvalidDeviceToken()
}

func IsTransientPushError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var apnsErr *APNsResponseError
	if errors.As(err, &apnsErr) {
		switch apnsErr.Reason {
		case "TooManyRequests", "ServiceUnavailable", "InternalServerError":
			return true
		}
		return apnsErr.StatusCode == 429 || apnsErr.StatusCode == 500 || apnsErr.StatusCode == 503
	}
	var fcmErr *FCMResponseError
	if errors.As(err, &fcmErr) {
		switch fcmErr.Reason {
		case "unavailable", "internal", "quota-exceeded":
			return true
		}
	}
	return false
}

func IsPermanentPushError(err error) bool {
	if err == nil || IsTransientPushError(err) {
		return false
	}
	if IsInvalidDeviceToken(err) {
		return true
	}
	var apnsErr *APNsResponseError
	if errors.As(err, &apnsErr) {
		return apnsErr.StatusCode >= 400 && apnsErr.StatusCode < 500
	}
	return false
}

func PushErrorReason(err error) string {
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

func isInvalidPushTokenError(err error) bool {
	return IsInvalidDeviceToken(err)
}

func pushErrorReason(err error) string {
	return PushErrorReason(err)
}

func isIOSPushToken(token repository.MobileDeviceToken) bool {
	return strings.EqualFold(strings.TrimSpace(token.Platform), "ios")
}

func marshalPushPayload(payload PushPayload) (string, error) {
	data, err := json.Marshal(payload.CustomPayload())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parsePushUserID(userID string) int {
	usrSeq, _ := strconv.Atoi(userID)
	return usrSeq
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

type pushPayloadJSON struct {
	EventType       string         `json:"event_type"`
	EventID         string         `json:"event_id"`
	TemplateKey     string         `json:"template_key"`
	TemplateVersion int            `json:"template_version"`
	TTLSec          int            `json:"ttl_sec"`
	CollapseKey     string         `json:"collapse_key"`
	UserID          string         `json:"user_id"`
	Args            map[string]any `json:"args"`
	DeepLink        string         `json:"deep_link"`
	SentAt          string         `json:"sent_at"`
}

func PushPayloadFromJSON(payloadJSON string) (PushPayload, error) {
	var stored pushPayloadJSON
	if err := json.Unmarshal([]byte(payloadJSON), &stored); err != nil {
		return PushPayload{}, err
	}
	sentAt, err := time.Parse(time.RFC3339, stored.SentAt)
	if err != nil {
		return PushPayload{}, err
	}
	return PushPayload{
		EventType:       stored.EventType,
		EventID:         stored.EventID,
		TemplateKey:     stored.TemplateKey,
		TemplateVersion: stored.TemplateVersion,
		TTLSec:          stored.TTLSec,
		CollapseKey:     stored.CollapseKey,
		UserID:          stored.UserID,
		Args:            stored.Args,
		DeepLink:        stored.DeepLink,
		SentAt:          sentAt.UTC(),
	}, nil
}

func PushNotificationFromOutboxJob(job repository.PushOutboxJob) (PushNotification, error) {
	payload, err := PushPayloadFromJSON(job.PayloadJSON)
	if err != nil {
		return PushNotification{}, err
	}
	bundleID := ""
	if job.BundleID.Valid {
		bundleID = job.BundleID.String
	}
	return PushNotification{
		DeviceToken:     job.DeviceToken,
		TokenID:         job.MDTSeq,
		Platform:        "ios",
		APNsEnvironment: normalizeAPNsEnvironment(job.APNsEnvironment),
		BundleID:        bundleID,
		Title:           job.Title,
		Body:            job.Body,
		Payload:         payload,
	}, nil
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
	return model.NormalizeAPNsEnvironment(env)
}

func hashPushToken(deviceToken string) string {
	sum := sha256.Sum256([]byte(deviceToken))
	return hex.EncodeToString(sum[:])[:16]
}

func HashPushTokenForLog(deviceToken string) string {
	return hashPushToken(deviceToken)
}
