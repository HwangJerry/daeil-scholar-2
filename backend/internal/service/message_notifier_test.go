package service

import (
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/realtime"
	"github.com/rs/zerolog"
)

func TestRealtimeMessageNotifierPublishesCanonicalMessageCreated(t *testing.T) {
	hub := realtime.NewHub(zerolog.Nop())
	sub := hub.Subscribe(202)
	defer hub.Unsubscribe(sub)
	notifier := NewRealtimeMessageNotifier(hub)
	accepted := &model.SendMessageResponse{
		MessageID: 9001,
		CreatedAt: "2026-07-28T01:00:00Z",
	}

	notifier.NotifyMessageReceived(202, 101, "예시 동문", accepted, "안녕하세요.")

	event := receiveNotifierEvent(t, sub)
	if event.Type != "message.created" {
		t.Fatalf("event type = %q, want message.created", event.Type)
	}
	payload := event.Payload.(map[string]any)
	if payload["messageId"] != int64(9001) || payload["conversationUserSeq"] != 101 {
		t.Fatalf("message identity payload = %#v", payload)
	}
	sender, ok := payload["sender"].(map[string]any)
	if !ok || sender["userSeq"] != 101 || sender["name"] != "예시 동문" {
		t.Fatalf("sender payload = %#v", payload["sender"])
	}
	if payload["preview"] != "안녕하세요." || payload["createdAt"] != "2026-07-28T01:00:00Z" {
		t.Fatalf("message content payload = %#v", payload)
	}
	if _, ok := payload["eventId"].(int64); !ok {
		t.Fatalf("eventId = %#v, want int64", payload["eventId"])
	}
	if len(payload) != 6 {
		t.Fatalf("payload keys = %#v, want canonical closed payload", payload)
	}
}

func TestRealtimeMessageNotifierPublishesConversationUpdatedToBothParticipants(t *testing.T) {
	hub := realtime.NewHub(zerolog.Nop())
	senderSub := hub.Subscribe(101)
	recipientSub := hub.Subscribe(202)
	defer hub.Unsubscribe(senderSub)
	defer hub.Unsubscribe(recipientSub)
	notifier := NewRealtimeMessageNotifier(hub)

	notifier.NotifyMessageSent(101, 202)

	assertConversationUpdatedEvent(t, receiveNotifierEvent(t, senderSub), 202)
	assertConversationUpdatedEvent(t, receiveNotifierEvent(t, recipientSub), 101)
}

func TestRealtimeMessageNotifierPublishesCanonicalMessageRead(t *testing.T) {
	hub := realtime.NewHub(zerolog.Nop())
	senderSub := hub.Subscribe(101)
	defer hub.Unsubscribe(senderSub)
	notifier := NewRealtimeMessageNotifier(hub)

	notifier.NotifyMessagesRead(101, 202, 9001, "2026-07-28T01:05:00Z")

	event := receiveNotifierEvent(t, senderSub)
	if event.Type != "message.read" {
		t.Fatalf("event type = %q, want message.read", event.Type)
	}
	payload := event.Payload.(map[string]any)
	if payload["conversationUserSeq"] != 202 || payload["throughMessageId"] != int64(9001) {
		t.Fatalf("read identity payload = %#v", payload)
	}
	if payload["readAt"] != "2026-07-28T01:05:00Z" {
		t.Fatalf("readAt = %#v", payload["readAt"])
	}
	if _, ok := payload["eventId"].(int64); !ok || len(payload) != 4 {
		t.Fatalf("payload = %#v, want closed message.read payload", payload)
	}
}

func assertConversationUpdatedEvent(t *testing.T, event realtime.Event, conversationUserSeq int) {
	t.Helper()
	if event.Type != "conversation.updated" {
		t.Fatalf("event type = %q, want conversation.updated", event.Type)
	}
	payload := event.Payload.(map[string]any)
	if payload["conversationUserSeq"] != conversationUserSeq {
		t.Fatalf("conversationUserSeq = %#v, want %d", payload["conversationUserSeq"], conversationUserSeq)
	}
	if _, ok := payload["eventId"].(int64); !ok || len(payload) != 2 {
		t.Fatalf("payload = %#v, want closed conversation.updated payload", payload)
	}
}

func receiveNotifierEvent(t *testing.T, sub *realtime.Subscriber) realtime.Event {
	t.Helper()
	select {
	case event := <-sub.Ch:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notifier event")
		return realtime.Event{}
	}
}
