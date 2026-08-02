package service

import (
	"context"
	"errors"
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
	notifier := NewPushDeliveryNotifier(store, provider, zerolog.Nop())
	notifier.deliver(context.Background(), pushDeliveryTestItem())

	if len(provider.calls) != 2 {
		t.Fatalf("provider calls = %d, want 2", len(provider.calls))
	}
	payload := provider.payloads[0]
	if payload.Type != "message" || payload.EventID != "9001" || payload.MessageID != "9001" ||
		payload.ConversationUserSeq != "101" || payload.SenderUserSeq != "101" ||
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
	NewPushDeliveryNotifier(store, provider, zerolog.Nop()).deliver(context.Background(), pushDeliveryTestItem())
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
	NewPushDeliveryNotifier(store, provider, zerolog.Nop()).deliver(context.Background(), pushDeliveryTestItem())
	if len(provider.payloads) != 1 || provider.payloads[0].Preview != "새 메시지가 도착했습니다." {
		t.Fatalf("payloads = %#v", provider.payloads)
	}
}

func TestPushDeliveryDeletesOnlyInvalidProviderToken(t *testing.T) {
	store := &pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{{Platform: "ios", DeviceToken: "invalid-token"}}}
	provider := &pushProviderStub{errors: []error{ErrPushInvalidToken}}
	NewPushDeliveryNotifier(store, provider, zerolog.Nop()).deliver(context.Background(), pushDeliveryTestItem())
	if len(store.deleted) != 1 || store.deleted[0].Platform != "ios" || store.deleted[0].DeviceToken != "invalid-token" {
		t.Fatalf("deleted = %#v", store.deleted)
	}
}

func TestPushDeliveryRetriesTransientProviderFailureTwice(t *testing.T) {
	store := &pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "token"}}}
	provider := &pushProviderStub{errors: []error{ErrPushTransient, ErrPushTransient, nil}}
	NewPushDeliveryNotifier(store, provider, zerolog.Nop()).deliver(context.Background(), pushDeliveryTestItem())
	if len(provider.calls) != 3 {
		t.Fatalf("provider calls = %d, want initial + 2 retries", len(provider.calls))
	}
}

func TestPushDeliveryQueueIsBoundedAndNonBlocking(t *testing.T) {
	notifier := NewPushDeliveryNotifier(&pushDeliveryStoreStub{}, &pushProviderStub{}, zerolog.Nop())
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
	NewPushDeliveryNotifier(store, provider, zerolog.Nop()).deliver(context.Background(), pushDeliveryTestItem())
	if len(provider.calls) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(provider.calls))
	}
}
