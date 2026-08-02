package push

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const firebaseMessagingScope = "https://www.googleapis.com/auth/firebase.messaging"

type Sender struct {
	fcm  *fcmSender
	apns *apnsSender
}

func NewSender(ctx context.Context, cfg config.PushConfig, client *http.Client) (*Sender, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, nil
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	fcm, err := newFCMSender(ctx, cfg.FCMProjectID, cfg.FCMCredentialsFile, client)
	if err != nil {
		return nil, fmt.Errorf("initialize FCM sender: %w", err)
	}
	apns, err := newAPNSSender(cfg.APNSTeamID, cfg.APNSKeyID, cfg.APNSPrivateKeyFile, client)
	if err != nil {
		return nil, fmt.Errorf("initialize APNs sender: %w", err)
	}
	return &Sender{fcm: fcm, apns: apns}, nil
}

func (s *Sender) Send(ctx context.Context, target model.PushDeliveryTarget, payload model.PushMessagePayload) error {
	switch target.Platform {
	case "android":
		return s.fcm.Send(ctx, target, payload)
	case "ios":
		return s.apns.Send(ctx, target, payload)
	default:
		return errors.New("unsupported push platform")
	}
}

type fcmSender struct {
	projectID   string
	client      *http.Client
	tokenSource oauth2.TokenSource
}

func newFCMSender(ctx context.Context, projectID, credentialsFile string, client *http.Client) (*fcmSender, error) {
	credentialsJSON, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, errors.New("read FCM credential file")
	}
	credentials, err := google.CredentialsFromJSON(ctx, credentialsJSON, firebaseMessagingScope)
	if err != nil {
		return nil, errors.New("parse FCM credentials")
	}
	return &fcmSender{projectID: projectID, client: client, tokenSource: oauth2.ReuseTokenSource(nil, credentials.TokenSource)}, nil
}

func (s *fcmSender) Send(ctx context.Context, target model.PushDeliveryTarget, payload model.PushMessagePayload) error {
	token, err := s.tokenSource.Token()
	if err != nil {
		return service.ErrPushTransient
	}
	body, err := json.Marshal(map[string]any{
		"message": map[string]any{
			"token":        target.DeviceToken,
			"notification": map[string]string{"title": payload.SenderName, "body": payload.Preview},
			"data":         payloadData(payload),
		},
	})
	if err != nil {
		return errors.New("encode FCM message")
	}
	endpoint := "https://fcm.googleapis.com/v1/projects/" + url.PathEscape(s.projectID) + "/messages:send"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return errors.New("build FCM request")
	}
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return service.ErrPushTransient
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	if fcmErrorCode(responseBody) == "UNREGISTERED" {
		return service.ErrPushInvalidToken
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		return service.ErrPushTransient
	}
	return errors.New("FCM rejected message")
}

func fcmErrorCode(body []byte) string {
	var value any
	if json.Unmarshal(body, &value) != nil {
		return ""
	}
	return findStringField(value, "errorCode")
}

func findStringField(value any, key string) string {
	switch typed := value.(type) {
	case map[string]any:
		if value, ok := typed[key].(string); ok {
			return value
		}
		for _, child := range typed {
			if found := findStringField(child, key); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range typed {
			if found := findStringField(child, key); found != "" {
				return found
			}
		}
	}
	return ""
}

type apnsSender struct {
	teamID string
	keyID  string
	key    *ecdsa.PrivateKey
	client *http.Client
	mu     sync.Mutex
	token  string
	expiry time.Time
}

func newAPNSSender(teamID, keyID, privateKeyFile string, client *http.Client) (*apnsSender, error) {
	keyPEM, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, errors.New("read APNs private key file")
	}
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("parse APNs private key")
	}
	key, err := parseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("parse APNs private key")
	}
	return &apnsSender{teamID: teamID, keyID: keyID, key: key, client: client}, nil
}

func parseECPrivateKey(der []byte) (*ecdsa.PrivateKey, error) {
	if value, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		key, ok := value.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("not EC key")
		}
		return key, nil
	}
	return x509.ParseECPrivateKey(der)
}

func (s *apnsSender) Send(ctx context.Context, target model.PushDeliveryTarget, payload model.PushMessagePayload) error {
	authToken, err := s.authToken()
	if err != nil {
		return errors.New("sign APNs token")
	}
	bodyValue := map[string]any{
		"aps": map[string]any{
			"alert": map[string]string{"title": payload.SenderName, "body": payload.Preview},
			"sound": "default",
		},
	}
	for key, value := range payloadData(payload) {
		bodyValue[key] = value
	}
	body, err := json.Marshal(bodyValue)
	if err != nil {
		return errors.New("encode APNs message")
	}
	host := "https://api.push.apple.com"
	if target.APNSEnvironment == "sandbox" {
		host = "https://api.sandbox.push.apple.com"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, host+"/3/device/"+url.PathEscape(target.DeviceToken), bytes.NewReader(body))
	if err != nil {
		return errors.New("build APNs request")
	}
	request.Header.Set("authorization", "bearer "+authToken)
	request.Header.Set("apns-topic", target.BundleID)
	request.Header.Set("apns-push-type", "alert")
	request.Header.Set("apns-priority", "10")
	response, err := s.client.Do(request)
	if err != nil {
		return service.ErrPushTransient
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if response.StatusCode == http.StatusOK {
		return nil
	}
	reason := apnsReason(responseBody)
	if response.StatusCode == http.StatusGone || reason == "BadDeviceToken" || reason == "DeviceTokenNotForTopic" || reason == "Unregistered" {
		return service.ErrPushInvalidToken
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		return service.ErrPushTransient
	}
	return errors.New("APNs rejected message")
}

func (s *apnsSender) authToken() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if s.token != "" && now.Before(s.expiry) {
		return s.token, nil
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"iss": s.teamID, "iat": now.Unix()})
	token.Header["kid"] = s.keyID
	signed, err := token.SignedString(s.key)
	if err != nil {
		return "", err
	}
	s.token = signed
	s.expiry = now.Add(50 * time.Minute)
	return signed, nil
}

func apnsReason(body []byte) string {
	var value struct {
		Reason string `json:"reason"`
	}
	if json.Unmarshal(body, &value) != nil {
		return ""
	}
	return value.Reason
}

func payloadData(payload model.PushMessagePayload) map[string]string {
	return map[string]string{
		"type":                payload.Type,
		"eventId":             payload.EventID,
		"messageId":           payload.MessageID,
		"conversationUserSeq": payload.ConversationUserSeq,
		"senderUserSeq":       payload.SenderUserSeq,
		"senderName":          payload.SenderName,
		"preview":             payload.Preview,
		"createdAt":           payload.CreatedAt,
	}
}
