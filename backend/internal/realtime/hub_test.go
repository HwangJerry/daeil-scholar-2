package realtime

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestHubPublishAssignsMonotonicEventIDs(t *testing.T) {
	hub := NewHub(zerolog.Nop())
	sub := hub.Subscribe(42)
	defer hub.Unsubscribe(sub)

	hub.Publish(42, Event{Type: "message.created", Payload: map[string]any{"messageId": int64(9001)}})
	hub.Publish(42, Event{Type: "conversation.updated", Payload: map[string]any{"conversationUserSeq": 202}})

	first := receiveEvent(t, sub)
	second := receiveEvent(t, sub)
	firstID, firstOK := first.Payload.(map[string]any)["eventId"].(int64)
	secondID, secondOK := second.Payload.(map[string]any)["eventId"].(int64)
	if !firstOK || firstID <= 0 {
		t.Fatalf("first eventId = %#v, want positive int64", first.Payload)
	}
	if !secondOK || secondID <= firstID {
		t.Fatalf("event IDs = (%d, %#v), want strictly increasing", firstID, second.Payload)
	}
	const maxJavaScriptSafeInteger = int64(1<<53 - 1)
	if secondID > maxJavaScriptSafeInteger {
		t.Fatalf("eventId = %d, exceeds JavaScript safe integer", secondID)
	}
}

func TestHubSubscribeReplaysEventsAfterLastEventID(t *testing.T) {
	hub := NewHub(zerolog.Nop())
	firstPayload := map[string]any{"messageId": int64(9001)}
	secondPayload := map[string]any{"messageId": int64(9002)}
	thirdPayload := map[string]any{"conversationUserSeq": 202}
	hub.Publish(42, Event{Type: "message.created", Payload: firstPayload})
	hub.Publish(42, Event{Type: "message.created", Payload: secondPayload})
	hub.Publish(42, Event{Type: "conversation.updated", Payload: thirdPayload})

	firstID := firstPayload["eventId"].(int64)
	sub := hub.Subscribe(42, firstID)
	defer hub.Unsubscribe(sub)

	second := receiveEvent(t, sub)
	third := receiveEvent(t, sub)
	if second.Payload.(map[string]any)["messageId"] != int64(9002) {
		t.Fatalf("second replay = %#v", second.Payload)
	}
	if third.Type != "conversation.updated" {
		t.Fatalf("third replay type = %q", third.Type)
	}
}

func TestHubRetainsOnlyLatest256EventsPerUser(t *testing.T) {
	hub := NewHub(zerolog.Nop())
	for messageID := int64(1); messageID <= 257; messageID++ {
		hub.Publish(42, Event{
			Type:    "message.created",
			Payload: map[string]any{"messageId": messageID},
		})
	}

	history := hub.history[42]
	if len(history) != 256 {
		t.Fatalf("history length = %d, want 256", len(history))
	}
	firstRetained := history[0].Payload.(map[string]any)["messageId"]
	if firstRetained != int64(2) {
		t.Fatalf("first retained messageId = %v, want 2", firstRetained)
	}
}

func TestHubReplayDoesNotBlockWhenMoreThanLiveBufferEventsArePending(t *testing.T) {
	hub := NewHub(zerolog.Nop())
	firstPayload := map[string]any{"messageId": int64(1)}
	hub.Publish(42, Event{Type: "message.created", Payload: firstPayload})
	for messageID := int64(2); messageID <= 10; messageID++ {
		hub.Publish(42, Event{Type: "message.created", Payload: map[string]any{"messageId": messageID}})
	}

	subscribed := make(chan *Subscriber, 1)
	go func() {
		subscribed <- hub.Subscribe(42, firstPayload["eventId"].(int64))
	}()

	select {
	case sub := <-subscribed:
		hub.Unsubscribe(sub)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subscribe blocked while enqueueing replay events")
	}
}

func TestHubSubscribeRaceWithPublishDoesNotLoseOrDuplicateEvent(t *testing.T) {
	hub := NewHub(zerolog.Nop())
	firstPayload := map[string]any{"messageId": int64(1)}
	hub.Publish(42, Event{Type: "message.created", Payload: firstPayload})

	hub.mu.Lock()
	subscribed := make(chan *Subscriber, 1)
	published := make(chan struct{})
	go func() {
		subscribed <- hub.Subscribe(42, firstPayload["eventId"].(int64))
	}()
	go func() {
		hub.Publish(42, Event{Type: "message.created", Payload: map[string]any{"messageId": int64(2)}})
		close(published)
	}()
	hub.mu.Unlock()

	sub := <-subscribed
	defer hub.Unsubscribe(sub)
	<-published
	event := receiveEvent(t, sub)
	if event.Payload.(map[string]any)["messageId"] != int64(2) {
		t.Fatalf("race event = %#v, want message 2", event.Payload)
	}
	select {
	case duplicate := <-sub.Ch:
		t.Fatalf("duplicate event after subscribe/publish race: %#v", duplicate)
	case <-time.After(50 * time.Millisecond):
	}
}

func receiveEvent(t *testing.T, sub *Subscriber) Event {
	t.Helper()
	select {
	case event := <-sub.Ch:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for realtime event")
		return Event{}
	}
}
