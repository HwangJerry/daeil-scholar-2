package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
)

type fakePushTokenStore struct {
	upserts         []model.PushDeviceRegistrationRequest
	deactivated     []string
	userTokens      []repository.MobileDeviceToken
	broadcastTokens []repository.MobileDeviceToken
	revokedTokens   []string
}

func (f *fakePushTokenStore) UpsertToken(_ int, req model.PushDeviceRegistrationRequest) error {
	f.upserts = append(f.upserts, req)
	return nil
}
func (f *fakePushTokenStore) DeactivateToken(_ int, deviceToken string) error {
	f.deactivated = append(f.deactivated, deviceToken)
	return nil
}
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

type fakePushProvider struct {
	err  error
	sent []PushNotification
}

func (f *fakePushProvider) SendPush(_ context.Context, notification PushNotification) error {
	f.sent = append(f.sent, notification)
	return f.err
}

type fakeInvalidTokenError struct{}

func (fakeInvalidTokenError) Error() string            { return "invalid token" }
func (fakeInvalidTokenError) InvalidDeviceToken() bool { return true }

func newInlinePushService(store *fakePushTokenStore, provider *fakePushProvider) *MobilePushService {
	return &MobilePushService{
		tokenRepo: store,
		provider:  provider,
		logger:    zerolog.Nop(),
		dispatch: func(job func()) {
			job()
		},
		now: func() time.Time {
			return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
		},
	}
}

func TestRegisterDeviceTokenUpsertsAPNsMetadata(t *testing.T) {
	store := &fakePushTokenStore{}
	svc := newInlinePushService(store, &fakePushProvider{})

	err := svc.RegisterDeviceToken(10, model.PushDeviceRegistrationRequest{
		Platform:        "IOS",
		DeviceToken:     " token-1 ",
		APNsEnvironment: "debug",
		BundleID:        "kr.dflh.saf",
		Locale:          "ko-KR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.upserts) != 1 {
		t.Fatalf("expected one upsert, got %d", len(store.upserts))
	}
	got := store.upserts[0]
	if got.Platform != "ios" || got.DeviceToken != "token-1" || got.APNsEnvironment != "sandbox" || got.BundleID != "kr.dflh.saf" {
		t.Fatalf("unexpected upsert payload: %#v", got)
	}
}

func TestRegisterDeviceTokenUpsertsAndroidPlatform(t *testing.T) {
	store := &fakePushTokenStore{}
	svc := newInlinePushService(store, &fakePushProvider{})

	err := svc.RegisterDeviceToken(10, model.PushDeviceRegistrationRequest{
		Platform:        " Android ",
		DeviceToken:     " fcm-token-1 ",
		APNsEnvironment: "sandbox",
		BundleID:        "kr.dflh.saf",
		Locale:          "ko-KR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.upserts) != 1 {
		t.Fatalf("expected one upsert, got %d", len(store.upserts))
	}
	got := store.upserts[0]
	if got.Platform != "android" || got.DeviceToken != "fcm-token-1" || got.APNsEnvironment != "" ||
		got.BundleID != "" || got.Locale != "ko-KR" {
		t.Fatalf("unexpected android upsert payload: %#v", got)
	}
}

func TestUnregisterDeviceTokenDeactivatesToken(t *testing.T) {
	store := &fakePushTokenStore{}
	svc := newInlinePushService(store, &fakePushProvider{})

	err := svc.UnregisterDeviceToken(10, " token-1 ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.deactivated) != 1 || store.deactivated[0] != "token-1" {
		t.Fatalf("expected token deactivated, got %#v", store.deactivated)
	}
}

func TestBuildMessageNewPayloadMatchesIOSContract(t *testing.T) {
	payload := BuildMessageNewPushPayload(777, 12345, 67890, time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC))
	custom := payload.CustomPayload()

	assertPushContractField(t, custom, "event_type")
	assertPushContractField(t, custom, "event")
	assertPushContractField(t, custom, "event_id")
	assertPushContractField(t, custom, "template_key")
	assertPushContractField(t, custom, "template_version")
	assertPushContractField(t, custom, "ttl_sec")
	assertPushContractField(t, custom, "collapse_key")
	assertPushContractField(t, custom, "user_id")
	assertPushContractField(t, custom, "args")
	assertPushContractField(t, custom, "deep_link")
	assertPushContractField(t, custom, "sent_at")

	if custom["event_type"] != PushEventMessageNew {
		t.Fatalf("unexpected event_type: %#v", custom["event_type"])
	}
	if custom["event"] != PushEventMessageNew {
		t.Fatalf("unexpected event: %#v", custom["event"])
	}
	if custom["event_id"] != "message.new:777:67890" {
		t.Fatalf("unexpected event_id: %#v", custom["event_id"])
	}
	if custom["template_key"] != "message_new_default" {
		t.Fatalf("unexpected template_key: %#v", custom["template_key"])
	}
	if custom["template_version"] != 1 {
		t.Fatalf("unexpected template_version: %#v", custom["template_version"])
	}
	if custom["collapse_key"] != "message.new:67890" {
		t.Fatalf("unexpected collapse_key: %#v", custom["collapse_key"])
	}
	if custom["user_id"] != "67890" {
		t.Fatalf("unexpected user_id: %#v", custom["user_id"])
	}
	if custom["deep_link"] != "/messages/12345" {
		t.Fatalf("unexpected deep_link: %#v", custom["deep_link"])
	}
	args := custom["args"].(map[string]any)
	if args["sender_seq"] != 12345 || args["recvr_seq"] != 67890 {
		t.Fatalf("unexpected args: %#v", args)
	}
	if custom["sent_at"] != "2026-07-02T00:00:00Z" {
		t.Fatalf("unexpected sent_at: %#v", custom["sent_at"])
	}
}

func TestBuildAdminNoticePayloadMatchesIOSContract(t *testing.T) {
	payload := BuildAdminNoticePushPayload(555, 67890, time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC))
	custom := payload.CustomPayload()

	assertPushContractField(t, custom, "event_type")
	assertPushContractField(t, custom, "event")
	assertPushContractField(t, custom, "event_id")
	assertPushContractField(t, custom, "template_key")
	assertPushContractField(t, custom, "template_version")
	assertPushContractField(t, custom, "ttl_sec")
	assertPushContractField(t, custom, "collapse_key")
	assertPushContractField(t, custom, "user_id")
	assertPushContractField(t, custom, "args")
	assertPushContractField(t, custom, "deep_link")
	assertPushContractField(t, custom, "sent_at")

	if custom["event_type"] != PushEventAdminNotice {
		t.Fatalf("unexpected event_type: %#v", custom["event_type"])
	}
	if custom["event"] != PushEventAdminNotice {
		t.Fatalf("unexpected event: %#v", custom["event"])
	}
	if custom["event_id"] != "admin.notice:555" {
		t.Fatalf("unexpected event_id: %#v", custom["event_id"])
	}
	if custom["template_key"] != "admin_notice_default" {
		t.Fatalf("unexpected template_key: %#v", custom["template_key"])
	}
	if custom["template_version"] != 1 {
		t.Fatalf("unexpected template_version: %#v", custom["template_version"])
	}
	if custom["collapse_key"] != "admin.notice:555" {
		t.Fatalf("unexpected collapse_key: %#v", custom["collapse_key"])
	}
	if custom["user_id"] != "67890" {
		t.Fatalf("unexpected user_id: %#v", custom["user_id"])
	}
	if custom["deep_link"] != "/feed/555" {
		t.Fatalf("unexpected deep_link: %#v", custom["deep_link"])
	}
	args := custom["args"].(map[string]any)
	if args["post_seq"] != 555 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestNotifyMessageReceivedSendsRecipientPayload(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{
			MDTSeq:          1,
			UsrSeq:          67890,
			Platform:        "ios",
			DeviceToken:     "token-1",
			APNsEnvironment: "sandbox",
			BundleID:        "kr.dflh.saf.debug",
		}},
	}
	provider := &fakePushProvider{}
	svc := newInlinePushService(store, provider)

	svc.NotifyMessageReceived(777, 67890, 12345, "홍길동", "private message body")

	if len(provider.sent) != 1 {
		t.Fatalf("expected 1 push, got %d", len(provider.sent))
	}
	push := provider.sent[0]
	if push.Body != "새로운 쪽지가 도착했습니다." {
		t.Fatalf("unexpected body: %q", push.Body)
	}
	if push.Payload.EventID != "message.new:777:67890" {
		t.Fatalf("unexpected event_id: %s", push.Payload.EventID)
	}
	if push.Payload.Args["sender_seq"] != 12345 || push.Payload.Args["recvr_seq"] != 67890 {
		t.Fatalf("unexpected args: %#v", push.Payload.Args)
	}
}

func TestNotifyMessageReceivedSendsAndroidPayload(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{
			MDTSeq:      1,
			UsrSeq:      67890,
			Platform:    "android",
			DeviceToken: "fcm-token-1",
		}},
	}
	provider := &fakePushProvider{}
	svc := newInlinePushService(store, provider)

	svc.NotifyMessageReceived(777, 67890, 12345, "홍길동", "private message body")

	if len(provider.sent) != 1 {
		t.Fatalf("expected 1 push, got %d", len(provider.sent))
	}
	push := provider.sent[0]
	if push.Platform != "android" || push.DeviceToken != "fcm-token-1" {
		t.Fatalf("unexpected android push target: %#v", push)
	}
	if push.Payload.EventType != PushEventMessageNew || push.Payload.Args["sender_seq"] != 12345 {
		t.Fatalf("unexpected android push payload: %#v", push.Payload)
	}
}

func TestNotifyPostPublishedCreatesPayloadPerRecipient(t *testing.T) {
	store := &fakePushTokenStore{
		broadcastTokens: []repository.MobileDeviceToken{
			{MDTSeq: 10, UsrSeq: 67890, Platform: "ios", DeviceToken: "token-10", APNsEnvironment: "production"},
			{MDTSeq: 11, UsrSeq: 67891, Platform: "ios", DeviceToken: "token-11", APNsEnvironment: "production"},
		},
	}
	provider := &fakePushProvider{}
	svc := newInlinePushService(store, provider)

	svc.NotifyPostPublished(99, 555, "공지")

	if len(provider.sent) != 2 {
		t.Fatalf("expected 2 pushes, got %d", len(provider.sent))
	}
	for i, wantUserID := range []string{"67890", "67891"} {
		push := provider.sent[i]
		if push.Payload.UserID != wantUserID {
			t.Fatalf("expected user_id %s for push %d, got %s", wantUserID, i, push.Payload.UserID)
		}
		if push.Payload.EventID != "admin.notice:555" || push.Payload.Args["post_seq"] != 555 {
			t.Fatalf("unexpected notice payload: %#v", push.Payload)
		}
	}
}

func TestNotifyMessageReceivedRevokesFCMInvalidToken(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{MDTSeq: 1, UsrSeq: 10, Platform: "android", DeviceToken: "dead-fcm-token"}},
	}
	provider := &fakePushProvider{err: &FCMResponseError{Reason: "registration-token-not-registered", invalidDeviceToken: true}}
	svc := newInlinePushService(store, provider)

	svc.NotifyMessageReceived(100, 10, 20, "Sender", "body")

	if len(store.revokedTokens) != 1 || store.revokedTokens[0] != "dead-fcm-token" {
		t.Fatalf("expected dead-fcm-token revoked, got %#v", store.revokedTokens)
	}
}

func TestNotifyMessageReceivedDoesNotRevokeFCMTransientError(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{MDTSeq: 1, UsrSeq: 10, Platform: "android", DeviceToken: "active-fcm-token"}},
	}
	provider := &fakePushProvider{err: &FCMResponseError{Reason: "unavailable", invalidDeviceToken: false}}
	svc := newInlinePushService(store, provider)

	svc.NotifyMessageReceived(100, 10, 20, "Sender", "body")

	if len(store.revokedTokens) != 0 {
		t.Fatalf("expected no revoked tokens, got %#v", store.revokedTokens)
	}
}

func TestNotifyMessageReceivedRevokesInvalidToken(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{MDTSeq: 1, UsrSeq: 10, Platform: "ios", DeviceToken: "dead-token", APNsEnvironment: "production"}},
	}
	provider := &fakePushProvider{err: fakeInvalidTokenError{}}
	svc := newInlinePushService(store, provider)

	svc.NotifyMessageReceived(100, 10, 20, "Sender", "body")

	if len(store.revokedTokens) != 1 || store.revokedTokens[0] != "dead-token" {
		t.Fatalf("expected dead-token revoked, got %#v", store.revokedTokens)
	}
}

func TestNotifyMessageReceivedDoesNotRevokeTransientError(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{MDTSeq: 1, UsrSeq: 10, Platform: "ios", DeviceToken: "active-token", APNsEnvironment: "production"}},
	}
	provider := &fakePushProvider{err: errors.New("temporary apns failure")}
	svc := newInlinePushService(store, provider)

	svc.NotifyMessageReceived(100, 10, 20, "Sender", "body")

	if len(store.revokedTokens) != 0 {
		t.Fatalf("expected no revoked tokens, got %#v", store.revokedTokens)
	}
}

func assertPushContractField(t *testing.T, payload map[string]any, field string) {
	t.Helper()
	if _, ok := payload[field]; !ok {
		t.Fatalf("missing payload field %q in %#v", field, payload)
	}
}
