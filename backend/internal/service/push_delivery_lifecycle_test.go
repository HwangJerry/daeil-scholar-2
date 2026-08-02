package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

type orderedPushProvider struct {
	mu  sync.Mutex
	ids []string
}

func (p *orderedPushProvider) Send(_ context.Context, _ model.PushDeliveryTarget, payload model.PushMessagePayload) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ids = append(p.ids, payload.MessageID)
	return nil
}

func TestPushDeliveryStopDrainsSameRecipientInOrder(t *testing.T) {
	store := &pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "token"}}}
	provider := &orderedPushProvider{}
	notifier := NewPushDeliveryNotifier(store, provider, zerolog.Nop())
	notifier.Start()
	for _, id := range []int64{9001, 9002} {
		notifier.NotifyMessageReceived(202, 101, "sender", &model.SendMessageResponse{MessageID: id}, "content")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := notifier.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if len(provider.ids) != 2 || provider.ids[0] != "9001" || provider.ids[1] != "9002" {
		t.Fatalf("delivery order = %#v", provider.ids)
	}
}
