// sms_sender.go — Outbound SMS abstraction and the no-delivery fallback used when
// no provider is configured. Vendor implementations live in their own files so the
// provider can be swapped (or a 본인확인 provider added) without touching callers.
package service

import (
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

// SMSSender delivers a single text message. Implementations must not log message
// bodies, which carry one-time verification codes.
type SMSSender interface {
	Send(model.SMSMessage) error
}

// NewSMSSender returns the sender matching the configured provider. When SMS is
// unconfigured it returns a sender that logs and drops the message, so dev and
// test environments run without an outbound vendor.
func NewSMSSender(cfg config.SMSConfig, logger zerolog.Logger) SMSSender {
	if !cfg.Configured() {
		return unconfiguredSMSSender{logger: logger}
	}
	switch cfg.Provider {
	case smsProviderNCP:
		return NewNCPSENSSender(cfg, logger)
	case smsProviderAligo:
		return NewAligoSMSSender(cfg, logger)
	default:
		logger.Warn().Str("provider", cfg.Provider).Msg("unknown SMS provider, skipping delivery")
		return unconfiguredSMSSender{logger: logger}
	}
}

// unconfiguredSMSSender drops messages and warns, mirroring how EmailService
// behaves when SMTP is absent.
type unconfiguredSMSSender struct {
	logger zerolog.Logger
}

func (s unconfiguredSMSSender) Send(msg model.SMSMessage) error {
	s.logger.Warn().Msg("SMS not configured, skipping verification code delivery")
	return nil
}
