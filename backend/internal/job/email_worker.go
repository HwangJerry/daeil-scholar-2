// email_worker.go — Background worker consuming email messages from a buffered channel
package job

import (
	"context"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

// EmailWorker reads EmailMessage values from a channel and delivers them
// via the EmailService in a dedicated goroutine.
type EmailWorker struct {
	queue  <-chan model.EmailMessage
	svc    *service.EmailService
	logger zerolog.Logger
	cancel context.CancelFunc
}

// NewEmailWorker creates an EmailWorker bound to the given channel and service.
func NewEmailWorker(queue <-chan model.EmailMessage, svc *service.EmailService, logger zerolog.Logger) *EmailWorker {
	return &EmailWorker{queue: queue, svc: svc, logger: logger}
}

// Start begins consuming messages from the queue in a background goroutine.
func (w *EmailWorker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.logger.Error().Interface("panic", r).Msg("email worker panicked")
			}
		}()
		w.logger.Info().Msg("email worker started")
		for {
			select {
			case <-ctx.Done():
				w.logger.Info().Msg("email worker stopped")
				return
			case msg, ok := <-w.queue:
				if !ok {
					return
				}
				if err := w.svc.Send(msg); err != nil {
					w.logger.Error().Err(err).Str("to", msg.To).Msg("email delivery failed")
				}
			}
		}
	}()
}

// Stop signals the background goroutine to exit.
func (w *EmailWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}
