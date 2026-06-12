package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
)

type fakePushTokenStore struct {
	userTokens      []repository.MobileDeviceToken
	broadcastTokens []repository.MobileDeviceToken
	revokedTokens   []string
}

func (f *fakePushTokenStore) UpsertToken(int, string, string, string) error { return nil }
func (f *fakePushTokenStore) DeactivateToken(int, string) error             { return nil }
func (f *fakePushTokenStore) RevokeToken(deviceToken string) error {
	f.revokedTokens = append(f.revokedTokens, deviceToken)
	return nil
}
func (f *fakePushTokenStore) GetActiveTokensByUser(int) ([]repository.MobileDeviceToken, error) {
	return f.userTokens, nil
}
func (f *fakePushTokenStore) GetActiveTokensForBroadcast(int) ([]repository.MobileDeviceToken, error) {
	return f.broadcastTokens, nil
}

type sentPush struct {
	deviceToken string
	title       string
	body        string
	data        map[string]any
}

type fakePushProvider struct {
	err  error
	sent []sentPush
}

func (f *fakePushProvider) SendPush(_ context.Context, deviceToken string, title string, body string, data map[string]any) error {
	f.sent = append(f.sent, sentPush{deviceToken: deviceToken, title: title, body: body, data: data})
	return f.err
}

type fakeInvalidTokenError struct{}

func (fakeInvalidTokenError) Error() string            { return "invalid token" }
func (fakeInvalidTokenError) InvalidDeviceToken() bool { return true }

func TestNotifyMessageReceivedDoesNotSendPII(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{UsrSeq: 10, DeviceToken: "token-1"}},
	}
	provider := &fakePushProvider{}
	svc := &MobilePushService{tokenRepo: store, provider: provider, logger: zerolog.Nop()}

	svc.NotifyMessageReceived(10, 20, "홍길동", "private message body")

	if len(provider.sent) != 1 {
		t.Fatalf("expected 1 push, got %d", len(provider.sent))
	}
	push := provider.sent[0]
	if push.body != "새로운 메시지가 도착했습니다." {
		t.Fatalf("expected generic message body, got %q", push.body)
	}
	args, ok := push.data["args"].(map[string]any)
	if !ok {
		t.Fatalf("expected args map, got %#v", push.data["args"])
	}
	if _, exists := args["sender_name"]; exists {
		t.Fatal("message push args must not include sender_name")
	}
	if _, exists := args["content_preview"]; exists {
		t.Fatal("message push args must not include content_preview")
	}
	if _, exists := args["sender_seq"]; exists {
		t.Fatal("message push args must not include sender_seq")
	}
	if push.data["deep_link"] != "/messages" {
		t.Fatalf("expected generic message deep_link, got %#v", push.data["deep_link"])
	}
}

func TestNotifyPostPublishedUsesRecipientID(t *testing.T) {
	store := &fakePushTokenStore{
		broadcastTokens: []repository.MobileDeviceToken{
			{UsrSeq: 10, DeviceToken: "token-10"},
			{UsrSeq: 11, DeviceToken: "token-11"},
		},
	}
	provider := &fakePushProvider{}
	svc := &MobilePushService{tokenRepo: store, provider: provider, logger: zerolog.Nop()}

	svc.NotifyPostPublished(99, 123, "공지")

	if len(provider.sent) != 2 {
		t.Fatalf("expected 2 pushes, got %d", len(provider.sent))
	}
	for i, wantUserID := range []int{10, 11} {
		gotUserID, ok := provider.sent[i].data["user_id"].(int)
		if !ok {
			t.Fatalf("expected int user_id for push %d, got %#v", i, provider.sent[i].data["user_id"])
		}
		if gotUserID != wantUserID {
			t.Fatalf("expected user_id %d for push %d, got %d", wantUserID, i, gotUserID)
		}
	}
}

func TestNotifyMessageReceivedRevokesInvalidToken(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{UsrSeq: 10, DeviceToken: "dead-token"}},
	}
	provider := &fakePushProvider{err: fakeInvalidTokenError{}}
	svc := &MobilePushService{tokenRepo: store, provider: provider, logger: zerolog.Nop()}

	svc.NotifyMessageReceived(10, 20, "Sender", "body")

	if len(store.revokedTokens) != 1 || store.revokedTokens[0] != "dead-token" {
		t.Fatalf("expected dead-token revoked, got %#v", store.revokedTokens)
	}
}

func TestNotifyMessageReceivedDoesNotRevokeTransientError(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{UsrSeq: 10, DeviceToken: "active-token"}},
	}
	provider := &fakePushProvider{err: errors.New("temporary apns failure")}
	svc := &MobilePushService{tokenRepo: store, provider: provider, logger: zerolog.Nop()}

	svc.NotifyMessageReceived(10, 20, "Sender", "body")

	if len(store.revokedTokens) != 0 {
		t.Fatalf("expected no revoked tokens, got %#v", store.revokedTokens)
	}
}
