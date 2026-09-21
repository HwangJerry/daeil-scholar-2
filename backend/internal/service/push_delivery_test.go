package service

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

type pushDeliveryStoreStub struct {
	preferences *model.PushPreferences
	targets     []model.PushDeliveryTarget
	deleted     []model.PushDeliveryTarget
}

func (s *pushDeliveryStoreStub) GetPreferences(int) (*model.PushPreferences, error) {
	return s.preferences, nil
}
func (s *pushDeliveryStoreStub) ListDevices(int) ([]model.PushDeliveryTarget, error) {
	return s.targets, nil
}
func (s *pushDeliveryStoreStub) DeleteDevice(platform, token string) error {
	s.deleted = append(s.deleted, model.PushDeliveryTarget{Platform: platform, DeviceToken: token})
	return nil
}

type pushProviderStub struct {
	calls    []model.PushDeliveryTarget
	payloads []model.PushMessagePayload
	errors   []error
}

func (s *pushProviderStub) Send(_ context.Context, target model.PushDeliveryTarget, payload model.PushMessagePayload) error {
	s.calls = append(s.calls, target)
	s.payloads = append(s.payloads, payload)
	if len(s.errors) == 0 {
		return nil
	}
	err := s.errors[0]
	s.errors = s.errors[1:]
	return err
}

func pushDeliveryTestItem() pushDeliveryItem {
	return pushDeliveryItem{
		recvrSeq: 202, senderSeq: 101, senderName: "예시 동문", content: "안녕하세요.",
		accepted: &model.SendMessageResponse{MessageID: 9001, CreatedAt: "2026-07-28T01:00:00Z"},
	}
}

func TestPushDeliveryFansOutCanonicalPayloadToEveryDevice(t *testing.T) {
	store := &pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{
		{Platform: "android", DeviceToken: "android-token"},
		{Platform: "ios", DeviceToken: "ios-token", APNSEnvironment: "sandbox", BundleID: "com.daeil.dflhsafv2"},
	}}
	provider := &pushProviderStub{}
	notifier := NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.deliver(context.Background(), pushDeliveryTestItem())

	if len(provider.calls) != 2 {
		t.Fatalf("provider calls = %d, want 2", len(provider.calls))
	}
	payload := provider.payloads[0]
	if payload.Type != "message" || payload.EventID != "9001" || payload.MessageID != "9001" ||
		payload.ConversationUserSeq != "101" || payload.SenderUserSeq != "101" ||
		payload.RecipientUserSeq != strconv.Itoa(pushDeliveryTestItem().recvrSeq) ||
		payload.SenderName != "예시 동문" || payload.Preview != "안녕하세요." || payload.CreatedAt != "2026-07-28T01:00:00Z" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestPushDeliverySuppressesDisabledMessageNotifications(t *testing.T) {
	store := &pushDeliveryStoreStub{
		preferences: &model.PushPreferences{MessageEnabled: false, MessagePreviewEnabled: true},
		targets:     []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "token"}},
	}
	provider := &pushProviderStub{}
	NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop()).deliver(context.Background(), pushDeliveryTestItem())
	if len(provider.calls) != 0 {
		t.Fatalf("provider calls = %d, want 0", len(provider.calls))
	}
}

func TestPushDeliveryMasksPreviewWhenDisabled(t *testing.T) {
	store := &pushDeliveryStoreStub{
		preferences: &model.PushPreferences{MessageEnabled: true, MessagePreviewEnabled: false},
		targets:     []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "token"}},
	}
	provider := &pushProviderStub{}
	NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop()).deliver(context.Background(), pushDeliveryTestItem())
	if len(provider.payloads) != 1 || provider.payloads[0].Preview != "새 메시지가 도착했습니다." {
		t.Fatalf("payloads = %#v", provider.payloads)
	}
}

func TestPushDeliveryDeletesOnlyInvalidProviderToken(t *testing.T) {
	store := &pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{{Platform: "ios", DeviceToken: "invalid-token"}}}
	provider := &pushProviderStub{errors: []error{ErrPushInvalidToken}}
	NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop()).deliver(context.Background(), pushDeliveryTestItem())
	if len(store.deleted) != 1 || store.deleted[0].Platform != "ios" || store.deleted[0].DeviceToken != "invalid-token" {
		t.Fatalf("deleted = %#v", store.deleted)
	}
}

func TestPushDeliveryRetriesTransientProviderFailureTwice(t *testing.T) {
	store := &pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "token"}}}
	provider := &pushProviderStub{errors: []error{ErrPushTransient, ErrPushTransient, nil}}
	NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop()).deliver(context.Background(), pushDeliveryTestItem())
	if len(provider.calls) != 3 {
		t.Fatalf("provider calls = %d, want initial + 2 retries", len(provider.calls))
	}
}

func TestPushDeliveryQueueIsBoundedAndNonBlocking(t *testing.T) {
	notifier := NewPushDeliveryNotifier(&pushDeliveryStoreStub{}, &pushProviderStub{}, NewTestNotificationTemplateService(nil), zerolog.Nop())
	for i := 0; i < pushShardCapacity+1; i++ {
		notifier.NotifyMessageReceived(202, 101, "sender", &model.SendMessageResponse{MessageID: int64(i + 1)}, "content")
	}
	shard := 202 % pushShardCount
	if got := len(notifier.shards[shard]); got != pushShardCapacity {
		t.Fatalf("queue length = %d, want %d", got, pushShardCapacity)
	}
}

func TestPushDeliveryDoesNotRetryPermanentFailure(t *testing.T) {
	store := &pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "token"}}}
	provider := &pushProviderStub{errors: []error{errors.New("permanent")}}
	NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop()).deliver(context.Background(), pushDeliveryTestItem())
	if len(provider.calls) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(provider.calls))
	}
}

// TestPushDeliveryRendersMessageTemplatesWithoutChangingDefaultText pins every
// field of the payload while no administrator has edited anything, so the
// templating step is provably byte-identical to the hardcoded text it replaced.
func TestPushDeliveryRendersMessageTemplatesWithoutChangingDefaultText(t *testing.T) {
	for _, test := range []struct {
		name        string
		preferences *model.PushPreferences
		want        model.PushMessagePayload
	}{
		{
			name:        "preview on",
			preferences: &model.PushPreferences{MessageEnabled: true, MessagePreviewEnabled: true},
			want: model.PushMessagePayload{
				Type: "message", EventID: "9001", MessageID: "9001",
				RecipientUserSeq: "202", ConversationUserSeq: "101", SenderUserSeq: "101",
				SenderName: "예시 동문", Title: "예시 동문",
				Body: "안녕하세요.", Preview: "안녕하세요.",
				CreatedAt:   "2026-07-28T01:00:00Z",
				TemplateKey: model.NotificationTemplateMessagePreviewOn,
			},
		},
		{
			name:        "preview off",
			preferences: &model.PushPreferences{MessageEnabled: true, MessagePreviewEnabled: false},
			want: model.PushMessagePayload{
				Type: "message", EventID: "9001", MessageID: "9001",
				RecipientUserSeq: "202", ConversationUserSeq: "101", SenderUserSeq: "101",
				SenderName: "예시 동문", Title: "예시 동문",
				Body: "새 메시지가 도착했습니다.", Preview: "새 메시지가 도착했습니다.",
				CreatedAt:   "2026-07-28T01:00:00Z",
				TemplateKey: model.NotificationTemplateMessagePreviewOff,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &pushDeliveryStoreStub{
				preferences: test.preferences,
				targets:     []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "token"}},
			}
			provider := &pushProviderStub{}
			NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop()).
				deliver(context.Background(), pushDeliveryTestItem())

			if provider.payloads[0] != test.want {
				t.Fatalf("payload = %#v, want %#v", provider.payloads[0], test.want)
			}
		})
	}
}

// TestPushDeliveryKeepsTheRoutingKeysWhenTemplatesAreEdited is the guard on the
// split between displayed text and data: editing the template must not change
// which conversation the apps attribute the push to, nor the snippet they list
// it by.
func TestPushDeliveryKeepsTheRoutingKeysWhenTemplatesAreEdited(t *testing.T) {
	templates := NewTestNotificationTemplateService(map[string]model.NotificationTemplate{
		model.NotificationTemplateMessagePreviewOn: {
			Key:     model.NotificationTemplateMessagePreviewOn,
			Channel: model.NotificationChannelPush,
			Title:   "대일외고 동문회",
			Body:    "{senderName}: {content}",
			Version: 7,
		},
	})
	store := &pushDeliveryStoreStub{
		preferences: &model.PushPreferences{MessageEnabled: true, MessagePreviewEnabled: true},
		targets:     []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "token"}},
	}
	provider := &pushProviderStub{}
	NewPushDeliveryNotifier(store, provider, templates, zerolog.Nop()).
		deliver(context.Background(), pushDeliveryTestItem())

	payload := provider.payloads[0]
	if payload.Title != "대일외고 동문회" || payload.Body != "예시 동문: 안녕하세요." {
		t.Fatalf("displayed text did not follow the edit: %#v", payload)
	}
	if payload.SenderName != "예시 동문" {
		t.Fatalf("senderName = %q, want the real sender", payload.SenderName)
	}
	if payload.Preview != "안녕하세요." {
		t.Fatalf("preview = %q, want the raw message snippet", payload.Preview)
	}
	if payload.TemplateVersion != 7 {
		t.Fatalf("template version = %d, want 7", payload.TemplateVersion)
	}
}

// TestPushDeliveryNeverLeaksContentWhenPreviewsAreDisabled is the privacy guard:
// no edit of the preview-off template can put the message content into either
// the displayed body or the data key.
func TestPushDeliveryNeverLeaksContentWhenPreviewsAreDisabled(t *testing.T) {
	templates := NewTestNotificationTemplateService(map[string]model.NotificationTemplate{
		model.NotificationTemplateMessagePreviewOff: {
			Key:     model.NotificationTemplateMessagePreviewOff,
			Channel: model.NotificationChannelPush,
			Title:   "대일외고 동문회",
			Body:    "읽지 않은 쪽지가 있습니다.",
			Version: 3,
		},
	})
	store := &pushDeliveryStoreStub{
		preferences: &model.PushPreferences{MessageEnabled: true, MessagePreviewEnabled: false},
		targets:     []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "token"}},
	}
	provider := &pushProviderStub{}
	NewPushDeliveryNotifier(store, provider, templates, zerolog.Nop()).
		deliver(context.Background(), pushDeliveryTestItem())

	payload := provider.payloads[0]
	if payload.Body != "읽지 않은 쪽지가 있습니다." || payload.Preview != "읽지 않은 쪽지가 있습니다." {
		t.Fatalf("hidden preview text not used: %#v", payload)
	}
	if payload.SenderName != "예시 동문" {
		t.Fatalf("senderName = %q, want the real sender", payload.SenderName)
	}
}

// messageGuardStoreStub counts the erasure guard's calls so the per-attempt
// cadence can be asserted.
type messageGuardStoreStub struct {
	pushDeliveryStoreStub
	checks    int
	available bool
}

func (s *messageGuardStoreStub) MessageStillAvailable(int, int, int64) (bool, error) {
	s.checks++
	return s.available, nil
}

// The erasure guard protects deleted content, so it must be re-asked before
// every send rather than once per queued item.
func TestPushDeliveryRechecksMessageAvailabilityOnEveryAttempt(t *testing.T) {
	store := &messageGuardStoreStub{
		pushDeliveryStoreStub: pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{
			{Platform: "android", DeviceToken: "android-token"},
			{Platform: "ios", DeviceToken: "ios-token", APNSEnvironment: "sandbox", BundleID: "com.daeil.dflhsafv2"},
		}},
		available: true,
	}
	// The first device exhausts all three attempts on transient errors, the
	// second succeeds immediately: four sends, and a guard call before each.
	provider := &pushProviderStub{errors: []error{ErrPushTransient, ErrPushTransient, nil, nil}}
	NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop()).
		deliver(context.Background(), pushDeliveryTestItem())

	if len(provider.calls) != 4 {
		t.Fatalf("provider calls = %d, want 4", len(provider.calls))
	}
	if store.checks != 4 {
		t.Fatalf("availability checks = %d, want one per attempt", store.checks)
	}
}

// An erased message stops the fan-out before the first send, including for a
// recipient's remaining devices.
func TestPushDeliveryStopsWhenMessageBecameUnavailable(t *testing.T) {
	store := &messageGuardStoreStub{
		pushDeliveryStoreStub: pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{
			{Platform: "android", DeviceToken: "android-token"},
			{Platform: "ios", DeviceToken: "ios-token", APNSEnvironment: "sandbox", BundleID: "com.daeil.dflhsafv2"},
		}},
		available: false,
	}
	provider := &pushProviderStub{}
	NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop()).
		deliver(context.Background(), pushDeliveryTestItem())

	if len(provider.calls) != 0 {
		t.Fatalf("an erased message was pushed to %d devices", len(provider.calls))
	}
	if store.checks != 1 {
		t.Fatalf("availability checks = %d, want the guard to stop delivery at once", store.checks)
	}
}
