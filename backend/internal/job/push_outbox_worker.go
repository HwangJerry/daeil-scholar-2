package job

import (
	"context"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

const maxPushOutboxErrorMessageLength = 1000

type PushOutboxWorkerConfig struct {
	BatchSize       int
	PollInterval    time.Duration
	MaxAttempts     int
	BaseBackoff     time.Duration
	MaxBackoff      time.Duration
	RecoveryTimeout time.Duration
	RequestTimeout  time.Duration
}

type PushOutboxStore interface {
	ClaimDue(ctx context.Context, batchSize int) ([]repository.PushOutboxJob, error)
	MarkSent(ctx context.Context, poSeq int) error
	MarkRetryScheduled(ctx context.Context, poSeq int, nextAttemptAt time.Time, errorCode string, errorMessage string) error
	MarkDead(ctx context.Context, poSeq int, errorCode string, errorMessage string) error
	ResetStuckProcessing(ctx context.Context, olderThan time.Duration) (int64, error)
}

type PushTokenRevoker interface {
	RevokeToken(deviceToken string) error
}

type PushPreferenceReader interface {
	GetPreferences(usrSeq int) (model.PushPreferences, error)
}

type PushOutboxWorker struct {
	outbox      PushOutboxStore
	tokens      PushTokenRevoker
	preferences PushPreferenceReader
	provider    service.MobilePushProvider
	cfg         PushOutboxWorkerConfig
	logger      zerolog.Logger
	cancel      context.CancelFunc
	now         func() time.Time
}

func NewPushOutboxWorker(
	outbox PushOutboxStore,
	tokens PushTokenRevoker,
	preferences PushPreferenceReader,
	provider service.MobilePushProvider,
	cfg PushOutboxWorkerConfig,
	logger zerolog.Logger,
) *PushOutboxWorker {
	return &PushOutboxWorker{
		outbox:      outbox,
		tokens:      tokens,
		preferences: preferences,
		provider:    provider,
		cfg:         normalizePushOutboxWorkerConfig(cfg),
		logger:      logger,
		now:         time.Now,
	}
}

func DefaultPushOutboxWorkerConfig() PushOutboxWorkerConfig {
	return PushOutboxWorkerConfig{
		BatchSize:       50,
		PollInterval:    5 * time.Second,
		MaxAttempts:     8,
		BaseBackoff:     30 * time.Second,
		MaxBackoff:      15 * time.Minute,
		RecoveryTimeout: 5 * time.Minute,
		RequestTimeout:  10 * time.Second,
	}
}

func (w *PushOutboxWorker) Start() {
	if w.outbox == nil {
		w.logger.Warn().Msg("push outbox worker disabled: repository is not configured")
		return
	}
	if w.provider == nil {
		w.logger.Warn().Msg("push outbox worker disabled: APNs provider is not configured")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	ticker := time.NewTicker(w.cfg.PollInterval)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.logger.Error().Interface("panic", r).Msg("push outbox worker panicked")
			}
		}()
		defer ticker.Stop()
		w.logger.Info().
			Int("batch_size", w.cfg.BatchSize).
			Dur("poll_interval", w.cfg.PollInterval).
			Int("max_attempts", w.cfg.MaxAttempts).
			Msg("push outbox worker started")
		for {
			if err := w.RunOnce(ctx); err != nil {
				w.logger.Error().Err(err).Msg("push outbox worker tick failed")
			}
			select {
			case <-ctx.Done():
				w.logger.Info().Msg("push outbox worker stopped")
				return
			case <-ticker.C:
			}
		}
	}()
}

func (w *PushOutboxWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *PushOutboxWorker) RunOnce(ctx context.Context) error {
	if w.outbox == nil || w.provider == nil {
		return nil
	}
	recovered, err := w.outbox.ResetStuckProcessing(ctx, w.cfg.RecoveryTimeout)
	if err != nil {
		return err
	}
	if recovered > 0 {
		w.logger.Warn().Int64("count", recovered).Msg("push outbox stuck jobs recovered")
	}
	jobs, err := w.outbox.ClaimDue(ctx, w.cfg.BatchSize)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		w.processJob(ctx, job)
	}
	return nil
}

func (w *PushOutboxWorker) processJob(parent context.Context, job repository.PushOutboxJob) {
	enabled, err := w.isPushEnabledForJob(job)
	if err != nil {
		w.scheduleRetryOrDead(parent, job, "PREFERENCE_LOOKUP_FAILED", err)
		return
	}
	if !enabled {
		w.markDead(parent, job, "PUSH_DISABLED_BY_USER", errPushDisabledByUser{})
		return
	}

	notification, err := service.PushNotificationFromOutboxJob(job)
	if err != nil {
		w.markDead(parent, job, "INVALID_PAYLOAD", err)
		return
	}
	ctx, cancel := context.WithTimeout(parent, w.cfg.RequestTimeout)
	defer cancel()
	err = w.provider.SendPush(ctx, notification)
	if err == nil {
		if markErr := w.outbox.MarkSent(parent, job.POSeq); markErr != nil {
			w.logger.Error().Err(markErr).Int("outbox_id", job.POSeq).Msg("push outbox mark sent failed")
			return
		}
		w.logJob(job, "SENT", "")
		return
	}

	reason := service.PushErrorReason(err)
	if service.IsInvalidDeviceToken(err) {
		if w.tokens != nil {
			if revokeErr := w.tokens.RevokeToken(job.DeviceToken); revokeErr != nil {
				w.logger.Error().
					Err(revokeErr).
					Int("outbox_id", job.POSeq).
					Int("token_id", job.MDTSeq).
					Str("token_hash", hashWorkerPushToken(job.DeviceToken)).
					Msg("push outbox invalid token revoke failed")
			}
		}
		w.markDead(parent, job, reasonOrDefault(reason, "INVALID_DEVICE_TOKEN"), err)
		return
	}
	if service.IsTransientPushError(err) {
		w.scheduleRetryOrDead(parent, job, reasonOrDefault(reason, "TRANSIENT_PUSH_ERROR"), err)
		return
	}

	w.logger.Error().
		Err(err).
		Int("outbox_id", job.POSeq).
		Str("event_type", job.EventType).
		Str("event_id", job.EventID).
		Int("user_id", job.UsrSeq).
		Int("token_id", job.MDTSeq).
		Str("token_hash", hashWorkerPushToken(job.DeviceToken)).
		Int("attempt_count", job.AttemptCount).
		Str("apns_reason", reason).
		Msg("push outbox permanent/config error")
	w.markDead(parent, job, reasonOrDefault(reason, "PERMANENT_PUSH_ERROR"), err)
}

func (w *PushOutboxWorker) isPushEnabledForJob(job repository.PushOutboxJob) (bool, error) {
	if w.preferences == nil {
		return true, nil
	}
	preferences, err := w.preferences.GetPreferences(job.UsrSeq)
	if err != nil {
		return false, err
	}
	switch job.EventType {
	case service.PushEventMessageNew:
		return preferences.MessageEnabled, nil
	case service.PushEventAdminNotice:
		return preferences.NoticeEnabled, nil
	default:
		return true, nil
	}
}

func (w *PushOutboxWorker) scheduleRetryOrDead(ctx context.Context, job repository.PushOutboxJob, code string, err error) {
	nextAttemptCount := job.AttemptCount + 1
	if nextAttemptCount >= w.cfg.MaxAttempts {
		w.markDead(ctx, job, code, err)
		return
	}
	nextAttemptAt := w.now().Add(w.backoffForAttempt(nextAttemptCount))
	if markErr := w.outbox.MarkRetryScheduled(ctx, job.POSeq, nextAttemptAt, code, truncatePushOutboxError(err.Error())); markErr != nil {
		w.logger.Error().Err(markErr).Int("outbox_id", job.POSeq).Msg("push outbox schedule retry failed")
		return
	}
	w.logJob(job, "FAILED", code)
}

func (w *PushOutboxWorker) markDead(ctx context.Context, job repository.PushOutboxJob, code string, err error) {
	if markErr := w.outbox.MarkDead(ctx, job.POSeq, code, truncatePushOutboxError(err.Error())); markErr != nil {
		w.logger.Error().Err(markErr).Int("outbox_id", job.POSeq).Msg("push outbox mark dead failed")
		return
	}
	w.logJob(job, "DEAD", code)
}

func (w *PushOutboxWorker) backoffForAttempt(attempt int) time.Duration {
	backoff := w.cfg.BaseBackoff
	for i := 1; i < attempt; i++ {
		backoff *= 2
		if backoff >= w.cfg.MaxBackoff {
			return w.cfg.MaxBackoff
		}
	}
	return backoff
}

func (w *PushOutboxWorker) logJob(job repository.PushOutboxJob, status string, reason string) {
	w.logger.Info().
		Int("outbox_id", job.POSeq).
		Str("event_type", job.EventType).
		Str("event_id", job.EventID).
		Int("user_id", job.UsrSeq).
		Int("token_id", job.MDTSeq).
		Str("token_hash", hashWorkerPushToken(job.DeviceToken)).
		Int("attempt_count", job.AttemptCount).
		Str("status", status).
		Str("apns_reason", reason).
		Msg("push outbox delivery result")
}

func normalizePushOutboxWorkerConfig(cfg PushOutboxWorkerConfig) PushOutboxWorkerConfig {
	defaults := DefaultPushOutboxWorkerConfig()
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaults.BatchSize
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaults.PollInterval
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = defaults.MaxAttempts
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = defaults.BaseBackoff
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = defaults.MaxBackoff
	}
	if cfg.RecoveryTimeout <= 0 {
		cfg.RecoveryTimeout = defaults.RecoveryTimeout
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaults.RequestTimeout
	}
	return cfg
}

func truncatePushOutboxError(message string) string {
	if len(message) <= maxPushOutboxErrorMessageLength {
		return message
	}
	return message[:maxPushOutboxErrorMessageLength]
}

func reasonOrDefault(reason string, fallback string) string {
	if reason != "" {
		return reason
	}
	return fallback
}

func hashWorkerPushToken(deviceToken string) string {
	return service.HashPushTokenForLog(deviceToken)
}

type errPushDisabledByUser struct{}

func (errPushDisabledByUser) Error() string {
	return "push disabled by user preference"
}
