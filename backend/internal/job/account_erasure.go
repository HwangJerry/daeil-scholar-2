// account_erasure.go — Cancellable automatic erasure queue runner.
package job

import (
	"context"
	"github.com/rs/zerolog"
	"time"
)

type erasureRunner interface{ RunOnce(context.Context) error }
type AccountErasureJob struct {
	runner erasureRunner
	logger zerolog.Logger
	cancel context.CancelFunc
	done   chan struct{}
}

func NewAccountErasureJob(r erasureRunner, l zerolog.Logger) *AccountErasureJob {
	return &AccountErasureJob{runner: r, logger: l}
}
func (j *AccountErasureJob) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	j.done = make(chan struct{})
	go func() {
		defer close(j.done)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			if err := j.runner.RunOnce(ctx); err != nil && ctx.Err() == nil {
				j.logger.Error().Msg("account erasure queue failed; retry scheduled")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
func (j *AccountErasureJob) Stop() {
	if j.cancel != nil {
		j.cancel()
		<-j.done
	}
}
