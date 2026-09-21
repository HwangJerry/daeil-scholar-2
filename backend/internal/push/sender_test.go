package push

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"golang.org/x/oauth2"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func providerPayload() model.PushMessagePayload {
	return model.PushMessagePayload{
		Type: "message", EventID: "9001", MessageID: "9001", ConversationUserSeq: "202",
		RecipientUserSeq: "303", SenderUserSeq: "202", SenderName: "예시 동문", Preview: "안녕하세요.", CreatedAt: "2026-07-28T01:00:00Z",
	}
}

func TestFCMSenderUsesHTTPV1CanonicalStringData(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/v1/projects/project-id/messages:send" {
			t.Fatalf("path = %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer access-token" {
			t.Fatalf("authorization header missing")
		}
		body, _ := io.ReadAll(request.Body)
		text := string(body)
		for _, fragment := range []string{`"token":"device-token"`, `"eventId":"9001"`, `"conversationUserSeq":"202"`, `"body":"안녕하세요."`} {
			if !strings.Contains(text, fragment) {
				t.Fatalf("body missing %s: %s", fragment, text)
			}
		}
		return response(http.StatusOK, `{}`), nil
	})}
	sender := &fcmSender{projectID: "project-id", client: client, tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "access-token"})}
	if err := sender.Send(context.Background(), model.PushDeliveryTarget{Platform: "android", DeviceToken: "device-token"}, providerPayload()); err != nil {
		t.Fatal(err)
	}
}

func TestFCMSenderClassifiesUnregisteredAndTransientWithoutExposingBody(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
		want   error
	}{
		{http.StatusNotFound, `{"error":{"details":[{"errorCode":"UNREGISTERED"}]}}`, service.ErrPushInvalidToken},
		{http.StatusTooManyRequests, `provider raw secret`, service.ErrPushTransient},
	} {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return response(test.status, test.body), nil })}
		sender := &fcmSender{projectID: "project", client: client, tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"})}
		err := sender.Send(context.Background(), model.PushDeliveryTarget{Platform: "android", DeviceToken: "device-token"}, providerPayload())
		if !errors.Is(err, test.want) || strings.Contains(err.Error(), test.body) {
			t.Fatalf("error = %v, want classification %v without raw body", err, test.want)
		}
	}
}

func TestAPNSSenderUsesSandboxTopicAndCanonicalStringData(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != "api.sandbox.push.apple.com" || request.URL.Path != "/3/device/device-token" {
			t.Fatalf("url = %s", request.URL.String())
		}
		if request.Header.Get("apns-topic") != "com.daeil.app" || request.Header.Get("apns-push-type") != "alert" {
			t.Fatalf("headers = %#v", request.Header)
		}
		body, _ := io.ReadAll(request.Body)
		text := string(body)
		for _, fragment := range []string{`"eventId":"9001"`, `"messageId":"9001"`, `"body":"안녕하세요."`} {
			if !strings.Contains(text, fragment) {
				t.Fatalf("body missing %s: %s", fragment, text)
			}
		}
		return response(http.StatusOK, ""), nil
	})}
	sender := &apnsSender{teamID: "team", keyID: "key", key: key, client: client}
	target := model.PushDeliveryTarget{Platform: "ios", DeviceToken: "device-token", APNSEnvironment: "sandbox", BundleID: "com.daeil.app"}
	if err := sender.Send(context.Background(), target, providerPayload()); err != nil {
		t.Fatal(err)
	}
}

func TestAPNSSenderClassifiesInvalidAndTransientResponses(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	for _, test := range []struct {
		status int
		body   string
		want   error
	}{
		{http.StatusBadRequest, `{"reason":"BadDeviceToken"}`, service.ErrPushInvalidToken},
		{http.StatusServiceUnavailable, `{"reason":"Shutdown"}`, service.ErrPushTransient},
	} {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return response(test.status, test.body), nil })}
		sender := &apnsSender{teamID: "team", keyID: "key", key: key, client: client}
		err := sender.Send(context.Background(), model.PushDeliveryTarget{Platform: "ios", DeviceToken: "token", BundleID: "bundle"}, providerPayload())
		if !errors.Is(err, test.want) || strings.Contains(err.Error(), test.body) {
			t.Fatalf("error = %v, want classification %v without raw body", err, test.want)
		}
	}
}

func TestPayloadDataCarriesAndroidEnvelopeForEveryType(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload model.PushMessagePayload
		want    map[string]string
		absent  []string
	}{
		{
			name:    "message",
			payload: model.PushMessagePayload{Type: "message", EventID: "9001", MessageID: "9001", RecipientUserSeq: "303", SenderUserSeq: "202", CreatedAt: "2026-07-28T01:00:00Z"},
			want:    map[string]string{"type": "message", "event_type": "message", "event_id": "9001", "user_id": "303", "ttl_sec": "86400", "sent_at": "1785200400", "sender_seq": "202", "recvr_seq": "303"},
			absent:  []string{},
		},
		{
			name:    "verification",
			payload: model.PushMessagePayload{Type: "verification.reviewed", EventID: "review-42", RecipientUserSeq: "42", CreatedAt: "2026-09-10T00:00:00Z"},
			want:    map[string]string{"type": "verification.reviewed", "event_type": "verification.reviewed", "event_id": "review-42", "user_id": "42", "ttl_sec": "86400", "sent_at": "1788998400", "recvr_seq": "42"},
			absent:  []string{"sender_seq"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := payloadData(test.payload)
			for key, want := range test.want {
				if data[key] != want {
					t.Fatalf("%s = %q, want %q", key, data[key], want)
				}
			}
			for _, key := range test.absent {
				if _, ok := data[key]; ok {
					t.Fatalf("%s present without a source value: %q", key, data[key])
				}
			}
			if _, ok := data["template_key"]; ok {
				t.Fatalf("template_key present without a template: %q", data["template_key"])
			}
			if _, ok := data["template_version"]; ok {
				t.Fatalf("template_version present without a template: %q", data["template_version"])
			}
		})
	}
}

func TestPayloadDataFallsBackToSendTimeForNonRFC3339CreatedAt(t *testing.T) {
	before := time.Now().Unix()
	data := payloadData(model.PushMessagePayload{Type: "message", EventID: "9001", RecipientUserSeq: "303", CreatedAt: "2026-07-28 01:00:00"})
	sentAt, err := strconv.ParseInt(data["sent_at"], 10, 64)
	if err != nil {
		t.Fatalf("sent_at = %q, want epoch seconds", data["sent_at"])
	}
	if sentAt < before || sentAt > time.Now().Unix() {
		t.Fatalf("sent_at = %d, want send time within [%d, now]", sentAt, before)
	}
	if data["ttl_sec"] != "86400" || data["event_type"] != "message" {
		t.Fatalf("android envelope incomplete: %#v", data)
	}
}

func TestPayloadDataOmitsRecipientKeysWhenUnset(t *testing.T) {
	data := payloadData(model.PushMessagePayload{Type: "message", EventID: "9001", CreatedAt: "2026-07-28T01:00:00Z"})
	for _, key := range []string{"user_id", "recvr_seq"} {
		if _, ok := data[key]; ok {
			t.Fatalf("%s present without a recipient: %q", key, data[key])
		}
	}
}

func TestPayloadDataCarriesTemplateIdentityWhenSet(t *testing.T) {
	payload := providerPayload()
	payload.TemplateKey = "message.new"
	payload.TemplateVersion = 3
	data := payloadData(payload)
	if data["template_key"] != "message.new" || data["template_version"] != "3" {
		t.Fatalf("template identity = %q/%q", data["template_key"], data["template_version"])
	}
}

func TestPayloadDataEmitsTemplateVersionAlongsideKeyEvenWhenZero(t *testing.T) {
	payload := providerPayload()
	payload.TemplateKey = "message.new"
	data := payloadData(payload)
	if data["template_key"] != "message.new" || data["template_version"] != "0" {
		t.Fatalf("template identity = %q/%q, want the pair emitted atomically", data["template_key"], data["template_version"])
	}
}
