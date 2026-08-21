// email_worker.go — Background worker consuming email messages from a buffered channel
package job

import (
	"sync"
	"time"

	"github.com/dflh-saf/backend/internal/maintenance"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

// EmailWorker reads EmailMessage values from a channel and delivers them
// via the EmailService in a dedicated goroutine.
type EmailWorker struct {
	queue           <-chan model.EmailMessage
	svc             EmailSender
	maintenanceGate *maintenance.Gate
	logger          zerolog.Logger
	stop            chan struct{}
	done            chan struct{}
	stopOnce        sync.Once
}

type EmailSender interface {
	Send(model.EmailMessage) error
}

// NewEmailWorker creates an EmailWorker bound to the given channel and service.
func NewEmailWorker(queue <-chan model.EmailMessage, svc EmailSender, maintenanceGate *maintenance.Gate, logger zerolog.Logger) *EmailWorker {
	return &EmailWorker{queue: queue, svc: svc, maintenanceGate: maintenanceGate, logger: logger}
}

// Start begins consuming messages from the queue in a background goroutine.
func (w *EmailWorker) Start() {
	w.stop = make(chan struct{})
	w.done = make(chan struct{})
	go func() {
		defer close(w.done)
		defer func() {
			if r := recover(); r != nil {
				w.logger.Error().Interface("panic", r).Msg("email worker panicked")
			}
		}()
		w.logger.Info().Msg("email worker started")
		for {
			if w.maintenanceGate.Active() {
				select {
				case <-w.stop:
					w.logger.Info().Msg("email worker stopped")
					return
				case <-time.After(100 * time.Millisecond):
					continue
				}
			}
			select {
			case <-w.stop:
				if !w.maintenanceGate.Active() {
					w.drainQueue()
				}
				w.logger.Info().Msg("email worker stopped")
				return
			case msg, ok := <-w.queue:
				if !ok {
					return
				}
				for !w.send(msg) {
					select {
					case <-w.stop:
						w.logger.Info().Msg("email worker stopped with a maintenance-blocked message")
						return
					case <-time.After(100 * time.Millisecond):
					}
				}
			}
		}
	}()
}

// Stop signals the background goroutine to exit.
func (w *EmailWorker) Stop() {
	if w.stop != nil {
		w.stopOnce.Do(func() { close(w.stop) })
	}
	if w.done != nil {
		<-w.done
	}
}

func (w *EmailWorker) drainQueue() {
	if w.svc == nil {
		return
	}
	for {
		select {
		case msg, ok := <-w.queue:
			if !ok {
				return
			}
			if !w.send(msg) {
				return
			}
		default:
			return
		}
	}
}

func (w *EmailWorker) send(msg model.EmailMessage) bool {
	if w.svc == nil {
		return true
	}
	lease, err := w.maintenanceGate.EnterBackground()
	if err != nil {
		return false
	}
	defer lease.Release()
	if err := w.svc.Send(msg); err != nil {
		w.logger.Error().Err(err).Str("to", msg.To).Msg("email delivery failed")
	}
	return true
}
