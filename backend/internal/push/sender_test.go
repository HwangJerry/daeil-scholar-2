package push

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

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
		SenderUserSeq: "202", SenderName: "예시 동문", Preview: "안녕하세요.", CreatedAt: "2026-07-28T01:00:00Z",
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
