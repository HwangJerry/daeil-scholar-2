package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

// The availability read can block on unrelated database work. Ownership may
// change while it waits, so the final session check must follow this read.
type availabilityBarrierPushStore struct {
	pushDeliveryStoreStub
	current           atomic.Bool
	availabilityCalls atomic.Int32
	pauseAt           int32
	entered           chan struct{}
	release           chan struct{}
}

func (s *availabilityBarrierPushStore) MessageStillAvailable(int, int, int64) (bool, error) {
	if s.availabilityCalls.Add(1) == s.pauseAt {
		s.entered <- struct{}{}
		<-s.release
	}
	return true, nil
}
func (s *availabilityBarrierPushStore) DeliveryTargetStillCurrent(int, model.PushDeliveryTarget) (bool, error) {
	return s.current.Load(), nil
}

func TestPushDeliveryFinalOwnerGuardAfterBlockedAvailability(t *testing.T) {
	for _, pauseAt := range []int32{1, 2} {
		t.Run(map[int32]string{1: "first-attempt", 2: "retry"}[pauseAt], func(t *testing.T) {
			store := &availabilityBarrierPushStore{pushDeliveryStoreStub: pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{{Platform: "ios", DeviceToken: "synthetic-old-device", SessionID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}}, pauseAt: pauseAt, entered: make(chan struct{}, 1), release: make(chan struct{})}
			store.current.Store(true)
			provider := &pushProviderStub{}
			if pauseAt == 2 {
				provider.errors = []error{ErrPushTransient}
			}
			notifier := NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop())
			done := make(chan struct{})
			go func() { defer close(done); notifier.deliver(context.Background(), pushDeliveryTestItem()) }()
			select {
			case <-store.entered:
			case <-time.After(2 * time.Second):
				close(store.release)
				t.Fatal("availability barrier was not reached")
			}
			// Models revoked old SID or a newer SID replacing this queued target.
			store.current.Store(false)
			close(store.release)
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("delivery did not leave barrier")
			}
			if got, want := len(provider.calls), int(pauseAt-1); got != want {
				t.Fatalf("provider started after old ownership changed: calls=%d want=%d", got, want)
			}
		})
	}
}
