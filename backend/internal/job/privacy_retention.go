// privacy_retention.go — Small, cancellable batches keep privacy retention deadlines enforceable.
package job

import (
	"context"
	"github.com/rs/zerolog"
	"time"
)

const privacyRetentionInterval = time.Minute
const privacyRetentionBatchSize = 100

type privacyRetentionRepository interface {
	PurgeExpiredPrivacyRecords(context.Context, int) error
}

type PrivacyRetentionJob struct {
	repo   privacyRetentionRepository
	logger zerolog.Logger
	cancel context.CancelFunc
	done   chan struct{}
}

func NewPrivacyRetentionJob(repo privacyRetentionRepository, logger zerolog.Logger) *PrivacyRetentionJob {
	return &PrivacyRetentionJob{repo: repo, logger: logger}
}

func (j *PrivacyRetentionJob) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel, j.done = cancel, make(chan struct{})
	go func() {
		defer close(j.done)
		ticker := time.NewTicker(privacyRetentionInterval)
		defer ticker.Stop()
		for {
			if err := j.repo.PurgeExpiredPrivacyRecords(ctx, privacyRetentionBatchSize); err != nil && ctx.Err() == nil {
				j.logger.Error().Err(err).Msg("privacy retention cleanup failed")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (j *PrivacyRetentionJob) Stop() {
	if j.cancel != nil {
		j.cancel()
	}
	if j.done != nil {
		<-j.done
	}
}
