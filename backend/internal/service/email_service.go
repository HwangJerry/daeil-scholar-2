// email_service.go — SMTP email delivery service wrapping net/smtp
package service

import (
	"fmt"
	"net/smtp"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

// EmailService sends emails via SMTP. When SMTP is not configured (Host is empty),
// it logs a warning and silently skips delivery.
type EmailService struct {
	cfg    config.SMTPConfig
	logger zerolog.Logger
}

// NewEmailService creates an EmailService with the given SMTP configuration.
func NewEmailService(cfg config.SMTPConfig, logger zerolog.Logger) *EmailService {
	return &EmailService{cfg: cfg, logger: logger}
}

// Send delivers a single email message via SMTP.
// Returns nil without sending if SMTP is not configured.
func (s *EmailService) Send(msg model.EmailMessage) error {
	if s.cfg.Host == "" {
		s.logger.Warn().
			Str("to", msg.To).
			Str("subject", msg.Subject).
			Msg("SMTP not configured, skipping email")
		return nil
	}

	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=utf-8\r\n\r\n",
		s.cfg.From, msg.To, msg.Subject,
	)
	body := headers + msg.Body

	auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
	addr := s.cfg.Host + ":" + s.cfg.Port

	err := smtp.SendMail(addr, auth, s.cfg.From, []string{msg.To}, []byte(body))
	if err != nil {
		s.logger.Error().Err(err).Str("to", msg.To).Msg("email send failed")
	}
	return err
}
