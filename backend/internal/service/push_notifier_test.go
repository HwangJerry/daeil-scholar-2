package service

import (
	"context"
	"encoding/json"
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

type fakePushOutboxStore struct {
	jobs []repository.PushOutboxInsert
	err  error
}

func (f *fakePushOutboxStore) Enqueue(_ context.Context, job repository.PushOutboxInsert) error {
	if f.err != nil {
		return f.err
	}
	f.jobs = append(f.jobs, job)
	return nil
}

type fakePushPreferenceStore struct {
	preferences map[int]model.PushPreferences
	upserts     []model.PushPreferences
}

func (f *fakePushPreferenceStore) GetPreferences(usrSeq int) (model.PushPreferences, error) {
	if f.preferences == nil {
		return model.DefaultPushPreferences(), nil
	}
	if preferences, ok := f.preferences[usrSeq]; ok {
		return preferences, nil
	}
	return model.DefaultPushPreferences(), nil
}

func (f *fakePushPreferenceStore) UpsertPreferences(_ int, preferences model.PushPreferences) (model.PushPreferences, error) {
	f.upserts = append(f.upserts, preferences)
	return preferences, nil
}

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

func newInlineOutboxPushService(store *fakePushTokenStore, outbox *fakePushOutboxStore, provider *fakePushProvider) *MobilePushService {
	svc := newInlinePushService(store, provider)
	svc.outbox = outbox
	return svc
}

func newInlinePushServiceWithPreferences(store *fakePushTokenStore, preferences *fakePushPreferenceStore, provider *fakePushProvider) *MobilePushService {
	svc := newInlinePushService(store, provider)
	svc.preferences = preferences
	return svc
}

func TestRegisterDeviceTokenUpsertsAPNsMetadata(t *testing.T) {
	store := &fakePushTokenStore{}
	svc := newInlinePushService(store, &fakePushProvider{})

	err := svc.RegisterDeviceToken(10, model.PushDeviceRegistrationRequest{
		Platform:        "IOS",
		DeviceToken:     " token-1 ",
		APNsEnvironment: "debug",
		BundleID:        "com.daeil.dflhsafv2",
		Locale:          "ko-KR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.upserts) != 1 {
		t.Fatalf("expected one upsert, got %d", len(store.upserts))
	}
	got := store.upserts[0]
	if got.Platform != "ios" || got.DeviceToken != "token-1" || got.APNsEnvironment != "sandbox" || got.BundleID != "com.daeil.dflhsafv2" {
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
		BundleID:        "com.daeil.dflhsafv2",
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
	if custom["template_key"] != "push.message.new" {
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
	if custom["template_key"] != "push.admin.notice" {
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
			BundleID:        "com.daeil.dflhsafv2",
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

func TestNotifyMessageReceivedSkipsWhenMessagePreferenceDisabled(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{
			MDTSeq:          1,
			UsrSeq:          67890,
			Platform:        "ios",
			DeviceToken:     "token-1",
			APNsEnvironment: "sandbox",
		}},
	}
	preferences := &fakePushPreferenceStore{
		preferences: map[int]model.PushPreferences{
			67890: {NoticeEnabled: true, MessageEnabled: false},
		},
	}
	provider := &fakePushProvider{}
	svc := newInlinePushServiceWithPreferences(store, preferences, provider)

	svc.NotifyMessageReceived(777, 67890, 12345, "홍길동", "private message body")

	if len(provider.sent) != 0 {
		t.Fatalf("expected push skipped by message preference, got %d sends", len(provider.sent))
	}
}

func TestNotifyMessageReceivedEnqueuesOutboxPerRecipientToken(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{
			{MDTSeq: 1, UsrSeq: 67890, Platform: "ios", DeviceToken: "token-1", APNsEnvironment: "sandbox", BundleID: "com.daeil.dflhsafv2"},
			{MDTSeq: 2, UsrSeq: 67890, Platform: "ios", DeviceToken: "token-2", APNsEnvironment: "production", BundleID: "com.daeil.dflhsafv2"},
		},
	}
	outbox := &fakePushOutboxStore{}
	provider := &fakePushProvider{}
	svc := newInlineOutboxPushService(store, outbox, provider)

	svc.NotifyMessageReceived(777, 67890, 12345, "홍길동", "private message body")

	if len(provider.sent) != 0 {
		t.Fatalf("expected provider not called during enqueue, got %d sends", len(provider.sent))
	}
	if len(outbox.jobs) != 2 {
		t.Fatalf("expected 2 outbox jobs, got %d", len(outbox.jobs))
	}
	for _, job := range outbox.jobs {
		if job.EventType != PushEventMessageNew || job.EventID != "message.new:777:67890" || job.UsrSeq != 67890 {
			t.Fatalf("unexpected outbox job: %#v", job)
		}
		payload := decodeStoredPushPayload(t, job.PayloadJSON)
		if payload["deep_link"] != "/messages/12345" || payload["user_id"] != "67890" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		args := payload["args"].(map[string]any)
		if args["sender_seq"] != float64(12345) || args["recvr_seq"] != float64(67890) {
			t.Fatalf("unexpected args: %#v", args)
		}
	}
}

func TestNotifyMessageReceivedOutboxDoesNotDependOnAsyncDispatch(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{
			MDTSeq:          1,
			UsrSeq:          67890,
			Platform:        "ios",
			DeviceToken:     "token-1",
			APNsEnvironment: "sandbox",
		}},
	}
	outbox := &fakePushOutboxStore{}
	svc := newInlineOutboxPushService(store, outbox, &fakePushProvider{})
	svc.dispatch = func(func()) {
		t.Fatalf("outbox enqueue must not depend on async dispatch")
	}

	svc.NotifyMessageReceived(777, 67890, 12345, "홍길동", "private message body")

	if len(outbox.jobs) != 1 {
		t.Fatalf("expected one durable outbox job, got %d", len(outbox.jobs))
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

func TestNotifyMessageReceivedKeepsAndroidDirectWhenOutboxEnabled(t *testing.T) {
	store := &fakePushTokenStore{
		userTokens: []repository.MobileDeviceToken{{
			MDTSeq:      1,
			UsrSeq:      67890,
			Platform:    "android",
			DeviceToken: "fcm-token-1",
		}},
	}
	outbox := &fakePushOutboxStore{}
	provider := &fakePushProvider{}
	svc := newInlineOutboxPushService(store, outbox, provider)

	svc.NotifyMessageReceived(777, 67890, 12345, "홍길동", "private message body")

	if len(outbox.jobs) != 0 {
		t.Fatalf("expected no iOS outbox jobs for android token, got %d", len(outbox.jobs))
	}
	if len(provider.sent) != 1 {
		t.Fatalf("expected android direct provider send, got %d", len(provider.sent))
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

func TestNotifyPostPublishedSkipsRecipientsWithNoticePreferenceDisabled(t *testing.T) {
	store := &fakePushTokenStore{
		broadcastTokens: []repository.MobileDeviceToken{
			{MDTSeq: 10, UsrSeq: 67890, Platform: "ios", DeviceToken: "token-10", APNsEnvironment: "production"},
			{MDTSeq: 11, UsrSeq: 67891, Platform: "ios", DeviceToken: "token-11", APNsEnvironment: "production"},
		},
	}
	preferences := &fakePushPreferenceStore{
		preferences: map[int]model.PushPreferences{
			67890: {NoticeEnabled: false, MessageEnabled: true},
			67891: {NoticeEnabled: true, MessageEnabled: true},
		},
	}
	provider := &fakePushProvider{}
	svc := newInlinePushServiceWithPreferences(store, preferences, provider)

	svc.NotifyPostPublished(99, 555, "공지")

	if len(provider.sent) != 1 {
		t.Fatalf("expected one notice push, got %d", len(provider.sent))
	}
	if provider.sent[0].Payload.UserID != "67891" {
		t.Fatalf("unexpected notice recipient: %#v", provider.sent[0])
	}
}

func TestNotifyPostPublishedEnqueuesOutboxWithRecipientUserPayload(t *testing.T) {
	store := &fakePushTokenStore{
		broadcastTokens: []repository.MobileDeviceToken{
			{MDTSeq: 10, UsrSeq: 67890, Platform: "ios", DeviceToken: "token-10", APNsEnvironment: "production"},
			{MDTSeq: 11, UsrSeq: 67891, Platform: "ios", DeviceToken: "token-11", APNsEnvironment: "production"},
		},
	}
	outbox := &fakePushOutboxStore{}
	svc := newInlineOutboxPushService(store, outbox, &fakePushProvider{})

	svc.NotifyPostPublished(99, 555, "공지")

	if len(outbox.jobs) != 2 {
		t.Fatalf("expected 2 outbox jobs, got %d", len(outbox.jobs))
	}
	for i, wantUserID := range []string{"67890", "67891"} {
		job := outbox.jobs[i]
		if job.EventType != PushEventAdminNotice || job.EventID != "admin.notice:555" {
			t.Fatalf("unexpected notice outbox job: %#v", job)
		}
		payload := decodeStoredPushPayload(t, job.PayloadJSON)
		if payload["user_id"] != wantUserID || payload["deep_link"] != "/feed/555" {
			t.Fatalf("unexpected payload for job %d: %#v", i, payload)
		}
		args := payload["args"].(map[string]any)
		if args["post_seq"] != float64(555) {
			t.Fatalf("unexpected args: %#v", args)
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

func decodeStoredPushPayload(t *testing.T, payloadJSON string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode payload json: %v", err)
	}
	return payload
}
