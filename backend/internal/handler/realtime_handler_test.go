package handler

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/realtime"
	"github.com/rs/zerolog"
)

func TestRealtimeStreamWritesMatchingSSEAndJSONEventIDs(t *testing.T) {
	hub := realtime.NewHub(zerolog.Nop())
	handler := NewRealtimeHandler(hub, zerolog.Nop())
	request := authRequest(http.MethodGet, "/api/messages/stream", nil)
	ctx, cancel := context.WithCancel(request.Context())
	defer cancel()
	request = request.WithContext(ctx)
	writer := newSSETestWriter()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.Stream(writer, request)
	}()

	writer.waitForFlush(t)
	payload := map[string]any{
		"messageId":           int64(9001),
		"conversationUserSeq": 202,
		"preview":             "안녕하세요.",
		"createdAt":           "2026-07-28T01:00:00Z",
	}
	hub.Publish(1, realtime.Event{Type: "message.created", Payload: payload})
	writer.waitForFlush(t)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after cancellation")
	}

	eventID := payload["eventId"].(int64)
	body := writer.String()
	wantIDLine := []byte("id: " + formatInt64(eventID) + "\n")
	if !bytes.Contains([]byte(body), wantIDLine) {
		t.Fatalf("stream body missing SSE id line %q:\n%s", wantIDLine, body)
	}
	if !bytes.Contains([]byte(body), []byte(`"eventId":`+formatInt64(eventID))) {
		t.Fatalf("stream body missing matching JSON eventId:\n%s", body)
	}
}

func TestRealtimeStreamReplaysEventsAfterLastEventID(t *testing.T) {
	hub := realtime.NewHub(zerolog.Nop())
	firstPayload := map[string]any{"messageId": int64(9001)}
	secondPayload := map[string]any{"messageId": int64(9002)}
	hub.Publish(1, realtime.Event{Type: "message.created", Payload: firstPayload})
	hub.Publish(1, realtime.Event{Type: "message.created", Payload: secondPayload})

	handler := NewRealtimeHandler(hub, zerolog.Nop())
	request := authRequest(http.MethodGet, "/api/messages/stream", nil)
	request.Header.Set("Last-Event-ID", formatInt64(firstPayload["eventId"].(int64)))
	ctx, cancel := context.WithCancel(request.Context())
	defer cancel()
	request = request.WithContext(ctx)
	writer := newSSETestWriter()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.Stream(writer, request)
	}()

	writer.waitForFlush(t) // ready
	writer.waitForFlush(t) // replay
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after cancellation")
	}

	body := writer.String()
	secondID := secondPayload["eventId"].(int64)
	if !bytes.Contains([]byte(body), []byte("id: "+formatInt64(secondID)+"\n")) {
		t.Fatalf("stream body missing replay event %d:\n%s", secondID, body)
	}
}

type sseTestWriter struct {
	mu      sync.Mutex
	header  http.Header
	body    bytes.Buffer
	flushed chan struct{}
}

func newSSETestWriter() *sseTestWriter {
	return &sseTestWriter{header: make(http.Header), flushed: make(chan struct{}, 8)}
}

func (w *sseTestWriter) Header() http.Header { return w.header }
func (w *sseTestWriter) WriteHeader(int)     {}
func (w *sseTestWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.Write(p)
}
func (w *sseTestWriter) Flush() {
	select {
	case w.flushed <- struct{}{}:
	default:
	}
}
func (w *sseTestWriter) waitForFlush(t *testing.T) {
	t.Helper()
	select {
	case <-w.flushed:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SSE flush")
	}
}
func (w *sseTestWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.String()
}

func formatInt64(value int64) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = digits[value%10]
		value /= 10
	}
	return string(buffer[index:])
}
