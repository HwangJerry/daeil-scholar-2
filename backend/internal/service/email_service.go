// email_service.go — SMTP email delivery service wrapping net/smtp
package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

const defaultSMTPTimeout = 15 * time.Second

// EmailService sends emails via SMTP. When SMTP is not configured (Host is empty),
// it logs a warning and silently skips delivery.
type EmailService struct {
	cfg     config.SMTPConfig
	logger  zerolog.Logger
	timeout time.Duration
}

// NewEmailService creates an EmailService with the given SMTP configuration.
func NewEmailService(cfg config.SMTPConfig, logger zerolog.Logger) *EmailService {
	return newEmailServiceWithTimeout(cfg, logger, defaultSMTPTimeout)
}

func newEmailServiceWithTimeout(cfg config.SMTPConfig, logger zerolog.Logger, timeout time.Duration) *EmailService {
	if timeout <= 0 {
		timeout = defaultSMTPTimeout
	}
	return &EmailService{cfg: cfg, logger: logger, timeout: timeout}
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

	addr := s.cfg.Host + ":" + s.cfg.Port
	err := s.sendSMTP(addr, msg.To, []byte(body))
	if err != nil {
		s.logger.Error().Err(err).Str("to", msg.To).Msg("email send failed")
	}
	return err
}

func (s *EmailService) sendSMTP(addr, recipient string, message []byte) error {
	conn, err := net.DialTimeout("tcp", addr, s.timeout)
	if err != nil {
		return err
	}
	if err := conn.SetDeadline(time.Now().Add(s.timeout)); err != nil {
		conn.Close()
		return err
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: s.cfg.Host,
		}); err != nil {
			return err
		}
	}
	if s.cfg.User != "" {
		if err := client.Auth(smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(s.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(recipient); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	// DATA's final 250 response is the SMTP delivery commit point. A missing
	// QUIT response after that point must not cause the worker to retry and
	// potentially deliver the same message twice.
	_ = client.Quit()
	return nil
}
