package job

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

const (
	blockedMessageCleanupBatchSize = 100
	blockedMessageCleanupInterval  = time.Minute
)

type blockedMessageCleanupRepository interface {
	DeleteExpiredSuppressedMessages(limit int) (int64, error)
}

// BlockedMessageCleanupJob removes only expired messages suppressed by a recipient block.
type BlockedMessageCleanupJob struct {
	repo     blockedMessageCleanupRepository
	logger   zerolog.Logger
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
}

func NewBlockedMessageCleanupJob(repo blockedMessageCleanupRepository, logger zerolog.Logger) *BlockedMessageCleanupJob {
	return &BlockedMessageCleanupJob{repo: repo, logger: logger, interval: blockedMessageCleanupInterval}
}

func (j *BlockedMessageCleanupJob) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	j.done = make(chan struct{})
	ticker := time.NewTicker(j.interval)
	go func() {
		defer close(j.done)
		defer func() {
			if recovered := recover(); recovered != nil {
				j.logger.Error().Interface("panic", recovered).Msg("blocked message cleanup job panicked")
			}
		}()
		defer ticker.Stop()

		j.runOnce()
		for {
			select {
			case <-ctx.Done():
				j.logger.Info().Msg("blocked message cleanup job stopped")
				return
			case <-ticker.C:
				j.runOnce()
			}
		}
	}()
}

func (j *BlockedMessageCleanupJob) Stop() {
	if j.cancel != nil {
		j.cancel()
	}
	if j.done != nil {
		<-j.done
	}
}

func (j *BlockedMessageCleanupJob) runOnce() {
	deleted, err := j.repo.DeleteExpiredSuppressedMessages(blockedMessageCleanupBatchSize)
	if err != nil {
		j.logger.Error().Err(err).Msg("blocked message cleanup failed")
		return
	}
	if deleted > 0 {
		j.logger.Info().Int64("count", deleted).Msg("expired blocked messages cleaned")
	}
}
