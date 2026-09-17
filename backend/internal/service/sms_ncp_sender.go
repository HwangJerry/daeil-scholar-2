// sms_ncp_sender.go — NAVER Cloud Platform SENS provider implementation of SMSSender.
package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

const (
	smsProviderNCP = "ncp"
	ncpSENSHost    = "https://sens.apigw.ntruss.com"
	ncpSENSTimeout = 10 * time.Second
	// SENS reports a queued send as HTTP-style 202 inside the response body.
	ncpSENSAcceptedStatus = "202"
	ncpSENSCountryCode    = "82"
)

// NCPSENSSender posts verification codes through the SENS SMS v2 API.
type NCPSENSSender struct {
	cfg    config.SMSConfig
	host   string
	client *http.Client
	logger zerolog.Logger
}

// NewNCPSENSSender creates an NCPSENSSender from the SMS configuration.
func NewNCPSENSSender(cfg config.SMSConfig, logger zerolog.Logger) *NCPSENSSender {
	return &NCPSENSSender{cfg: cfg, host: ncpSENSHost, client: &http.Client{Timeout: ncpSENSTimeout}, logger: logger}
}

type ncpSENSRecipient struct {
	To string `json:"to"`
}

type ncpSENSRequest struct {
	Type        string             `json:"type"`
	ContentType string             `json:"contentType"`
	CountryCode string             `json:"countryCode"`
	From        string             `json:"from"`
	Content     string             `json:"content"`
	Messages    []ncpSENSRecipient `json:"messages"`
}

type ncpSENSResponse struct {
	RequestID  string `json:"requestId"`
	StatusCode string `json:"statusCode"`
	StatusName string `json:"statusName"`
}

// Send delivers one message. The body is never logged because it carries a one-time code.
func (s *NCPSENSSender) Send(msg model.SMSMessage) error {
	path := "/sms/v2/services/" + s.cfg.NCPServiceID + "/messages"
	payload, err := json.Marshal(ncpSENSRequest{
		Type:        "SMS",
		ContentType: "COMM",
		CountryCode: ncpSENSCountryCode,
		From:        s.cfg.Sender,
		Content:     msg.Body,
		Messages:    []ncpSENSRecipient{{To: msg.To}},
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, s.host+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-ncp-apigw-timestamp", timestamp)
	request.Header.Set("x-ncp-iam-access-key", s.cfg.NCPAccessKey)
	request.Header.Set("x-ncp-apigw-signature-v2", s.signature(http.MethodPost, path, timestamp))

	response, err := s.client.Do(request)
	if err != nil {
		s.logger.Error().Err(err).Msg("ncp sens request failed")
		return err
	}
	defer response.Body.Close()

	var decoded ncpSENSResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		s.logger.Error().Err(err).Int("status", response.StatusCode).Msg("ncp sens response decode failed")
		return err
	}
	if decoded.StatusCode != ncpSENSAcceptedStatus {
		s.logger.Error().
			Str("statusCode", decoded.StatusCode).
			Str("statusName", decoded.StatusName).
			Str("requestId", decoded.RequestID).
			Msg("ncp sens delivery rejected")
		return fmt.Errorf("ncp sens: statusCode %s", decoded.StatusCode)
	}
	return nil
}

// signature builds x-ncp-apigw-signature-v2: base64(HmacSHA256(secretKey,
// "METHOD URI\ntimestamp\naccessKey")). The timestamp must be the same value sent
// in x-ncp-apigw-timestamp or the gateway rejects the request.
func (s *NCPSENSSender) signature(method, uri, timestamp string) string {
	message := method + " " + uri + "\n" + timestamp + "\n" + s.cfg.NCPAccessKey
	mac := hmac.New(sha256.New, []byte(s.cfg.NCPSecretKey))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
