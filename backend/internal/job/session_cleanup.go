package job

import (
	"context"
	"time"

	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
)

type SessionCleanupJob struct {
	repo   *repository.SessionRepository
	logger zerolog.Logger
	cancel context.CancelFunc
}

func NewSessionCleanupJob(repo *repository.SessionRepository, logger zerolog.Logger) *SessionCleanupJob {
	return &SessionCleanupJob{repo: repo, logger: logger}
}

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
				deleted, err := j.repo.DeleteExpiredSessions()
				if err != nil {
					j.logger.Error().Err(err).Msg("session cleanup failed")
					continue
				}
				j.logger.Info().Int64("count", deleted).Msg("expired sessions cleaned")
			}
		}
	}()
}

func (j *SessionCleanupJob) Stop() {
	if j.cancel != nil {
		j.cancel()
	}
}
