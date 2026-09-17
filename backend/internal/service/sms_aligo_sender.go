// sms_aligo_sender.go — Aligo (알리고) SMS provider implementation of SMSSender.
package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

const (
	smsProviderAligo  = "aligo"
	aligoSendEndpoint = "https://apis.aligo.in/send/"
	aligoTimeout      = 10 * time.Second
	// Aligo reports application-level failures inside a 200 response body.
	aligoSuccessCode = 1
)

// AligoSMSSender posts verification codes through the Aligo REST API.
type AligoSMSSender struct {
	cfg    config.SMSConfig
	client *http.Client
	logger zerolog.Logger
}

// NewAligoSMSSender creates an AligoSMSSender from the SMS configuration.
func NewAligoSMSSender(cfg config.SMSConfig, logger zerolog.Logger) *AligoSMSSender {
	return &AligoSMSSender{cfg: cfg, client: &http.Client{Timeout: aligoTimeout}, logger: logger}
}

type aligoResponse struct {
	ResultCode int    `json:"result_code"`
	Message    string `json:"message"`
}

// Send delivers one message. The body is never logged because it carries a one-time code.
func (s *AligoSMSSender) Send(msg model.SMSMessage) error {
	form := url.Values{
		"key":      {s.cfg.APIKey},
		"user_id":  {s.cfg.UserID},
		"sender":   {s.cfg.Sender},
		"receiver": {msg.To},
		"msg":      {msg.Body},
		"msg_type": {"SMS"},
	}

	response, err := s.client.PostForm(aligoSendEndpoint, form)
	if err != nil {
		s.logger.Error().Err(err).Msg("aligo sms request failed")
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		s.logger.Error().Int("status", response.StatusCode).Msg("aligo sms rejected")
		return fmt.Errorf("aligo sms: unexpected status %d", response.StatusCode)
	}

	var decoded aligoResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		s.logger.Error().Err(err).Msg("aligo sms response decode failed")
		return err
	}
	if decoded.ResultCode != aligoSuccessCode {
		s.logger.Error().
			Int("resultCode", decoded.ResultCode).
			Str("providerMessage", strings.TrimSpace(decoded.Message)).
			Msg("aligo sms delivery failed")
		return fmt.Errorf("aligo sms: result_code %d", decoded.ResultCode)
	}
	return nil
}
