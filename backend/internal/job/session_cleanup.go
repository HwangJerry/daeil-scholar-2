// session_cleanup.go — Hourly background job for expired session and token cleanup
package job

import (
	"context"
	"time"

	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
)

// SessionCleanupJob periodically removes expired sessions and password reset tokens.
type SessionCleanupJob struct {
	sessionRepo       *repository.SessionRepository
	passwordResetRepo *repository.PasswordResetRepository
	logger            zerolog.Logger
	cancel            context.CancelFunc
}

// NewSessionCleanupJob creates a SessionCleanupJob with all required repositories.
func NewSessionCleanupJob(
	sessionRepo *repository.SessionRepository,
	passwordResetRepo *repository.PasswordResetRepository,
	logger zerolog.Logger,
) *SessionCleanupJob {
	return &SessionCleanupJob{
		sessionRepo:       sessionRepo,
		passwordResetRepo: passwordResetRepo,
		logger:            logger,
	}
}

// Start begins the hourly cleanup loop in a background goroutine.
func (j *SessionCleanupJob) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	ticker := time.NewTicker(time.Hour)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				j.logger.Error().Interface("panic", r).Msg("session cleanup job panicked")
			}
		}()
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				j.logger.Info().Msg("session cleanup job stopped")
				return
			case <-ticker.C:
				j.cleanSessions()
				j.cleanExpiredTokens()
			}
		}
	}()
}

// Stop signals the background goroutine to exit.
func (j *SessionCleanupJob) Stop() {
	if j.cancel != nil {
		j.cancel()
	}
}

func (j *SessionCleanupJob) cleanSessions() {
	deleted, err := j.sessionRepo.DeleteExpiredSessions()
	if err != nil {
		j.logger.Error().Err(err).Msg("session cleanup failed")
		return
	}
	j.logger.Info().Int64("count", deleted).Msg("expired sessions cleaned")
}

func (j *SessionCleanupJob) cleanExpiredTokens() {
	if j.passwordResetRepo == nil {
		return
	}
	deleted, err := j.passwordResetRepo.DeleteExpiredTokens()
	if err != nil {
		j.logger.Error().Err(err).Msg("password reset token cleanup failed")
		return
	}
	if deleted > 0 {
		j.logger.Info().Int64("count", deleted).Msg("expired password reset tokens cleaned")
	}
}
