package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/golden"
)

const realtimeFrameTimeout = 10 * time.Second

var goldenRecipient = goldenMemberSeed{
	seq: goldenMemberID + 1, usrID: "golden_recipient", name: "합성 수신자", phone: "01000000003", email: "recipient@example.test",
}

// sseFrame is one dispatched SSE frame exactly as written by the server.
type sseFrame struct {
	raw   string // Frame lines including the terminating blank line.
	id    int64  // Zero when the frame has no id: line (the ready frame).
	event string
	data  string
}

// TestGoldenRealtimeMessageEvents captures the SSE frames both apps decode, through
// GET /api/messages/stream on the real router: A sends B a message, then B reads it.
func TestGoldenRealtimeMessageEvents(t *testing.T) {
	s := newGoldenServer(t)
	sender, recipient := defaultGoldenMember, goldenRecipient
	seedGoldenMemberAs(t, s.db, sender)
	seedGoldenMemberAs(t, s.db, recipient)
	_, senderSession := s.loginAs(t, sender, "android")
	_, recipientSession := s.loginAs(t, recipient, "ios")

	senderStream := s.openStream(t, senderSession.AccessToken, "android")
	recipientStream := s.openStream(t, recipientSession.AccessToken, "ios")
	// Subscription happens before the ready frame is written, so events cannot be missed.
	ready := senderStream.next(t, "ready")
	recipientStream.next(t, "ready")

	body := s.request(t, http.MethodPost, "/api/messages", map[string]any{
		"userSeq": recipient.seq, "clientMessageId": "ts16-synthetic-client-message", "content": "합성 실시간 쪽지",
	}, senderSession.AccessToken, "android", "100", http.StatusOK)
	accepted := decodeGolden[model.SendMessageResponse](t, body)
	if accepted.MessageID <= 0 || accepted.CreatedAt == "" {
		t.Fatalf("send must accept a message: %s", body)
	}
	created := recipientStream.next(t, "message.created")
	recipientUpdated := recipientStream.next(t, "conversation.updated")
	senderUpdated := senderStream.next(t, "conversation.updated")

	s.request(t, http.MethodPut, fmt.Sprintf("/api/messages/conversations/%d/read", sender.seq), map[string]any{
		"throughMessageId": accepted.MessageID,
	}, recipientSession.AccessToken, "ios", "100", http.StatusNoContent)
	read := senderStream.next(t, "message.read")

	assertRealtimePayload(t, created, map[string]any{
		"messageId": float64(accepted.MessageID), "conversationUserSeq": float64(sender.seq), "preview": "합성 실시간 쪽지",
	})
	if payload := decodeRealtimeData(t, created); fmt.Sprint(payload["sender"]) != fmt.Sprint(map[string]any{"userSeq": float64(sender.seq), "name": sender.name}) {
		t.Fatalf("message.created must carry the sender: %s", created.data)
	}
	assertRealtimePayload(t, recipientUpdated, map[string]any{"conversationUserSeq": float64(sender.seq)})
	assertRealtimePayload(t, senderUpdated, map[string]any{"conversationUserSeq": float64(recipient.seq)})
	assertRealtimePayload(t, read, map[string]any{"conversationUserSeq": float64(recipient.seq), "throughMessageId": float64(accepted.MessageID)})
	if ready.raw != "event: ready\ndata: {\"ok\":true}\n\n" {
		t.Fatalf("unexpected ready frame %q", ready.raw)
	}

	// Event IDs start at the server clock; renumber them from 1 in publish order so
	// fixtures keep numeric IDs and the relative order the hub assigned.
	frames := []sseFrame{created, senderUpdated, recipientUpdated, read}
	baseID := created.id
	for _, frame := range frames {
		baseID = min(baseID, frame.id)
	}
	for name, frame := range map[string]sseFrame{
		"realtime_ready":                         ready,
		"realtime_message_created":               created,
		"realtime_conversation_updated_sender":    senderUpdated,
		"realtime_conversation_updated_recipient": recipientUpdated,
		"realtime_message_read":                  read,
	} {
		golden.Assert(t, name, realtimeGolden(t, frame, baseID-1))
	}
}

// realtimeGolden builds {event, data, raw} with the eventId renumbered and data
// normalized by the shared helper (timestamps become <timestamp>), so raw and data agree.
func realtimeGolden(t *testing.T, frame sseFrame, idOffset int64) []byte {
	t.Helper()
	var data map[string]any
	decoder := json.NewDecoder(strings.NewReader(frame.data))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		t.Fatal(err)
	}
	var lines []string
	if frame.id != 0 {
		id := frame.id - idOffset
		data["eventId"] = id
		lines = append(lines, "id: "+strconv.FormatInt(id, 10))
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := golden.Normalize(encoded)
	if err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, normalized); err != nil {
		t.Fatal(err)
	}
	lines = append(lines, "event: "+frame.event, "data: "+compact.String())
	envelope, err := json.Marshal(map[string]any{
		"event": frame.event,
		"data":  json.RawMessage(normalized),
		"raw":   strings.Join(lines, "\n") + "\n\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}

func assertRealtimePayload(t *testing.T, frame sseFrame, want map[string]any) {
	t.Helper()
	payload := decodeRealtimeData(t, frame)
	for key, value := range want {
		if payload[key] != value {
			t.Fatalf("%s %s = %v, want %v: %s", frame.event, key, payload[key], value, frame.data)
		}
	}
	if eventID, ok := payload["eventId"].(float64); !ok || int64(eventID) != frame.id || frame.id <= 0 {
		t.Fatalf("%s data eventId must match the id: line %d: %s", frame.event, frame.id, frame.data)
	}
	wantRaw := fmt.Sprintf("id: %d\nevent: %s\ndata: %s\n\n", frame.id, frame.event, frame.data)
	if frame.raw != wantRaw {
		t.Fatalf("frame must be exactly id/event/data lines: %q", frame.raw)
	}
}

func decodeRealtimeData(t *testing.T, frame sseFrame) map[string]any {
	t.Helper()
	return decodeGolden[map[string]any](t, []byte(frame.data))
}

type sseStream struct {
	frames chan sseFrame
	errs   chan error
}

func (s *goldenServer) openStream(t *testing.T, accessToken, platform string) *sseStream {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.server.URL+"/api/messages/stream", nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")
	setGoldenClientHeaders(req, accessToken, platform, "100")
	// No client timeout: the stream stays open until cleanup cancels it.
	resp, err := (&http.Client{Transport: s.client.Transport}).Do(req)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		resp.Body.Close()
	})
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("stream status %d, content type %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	stream := &sseStream{frames: make(chan sseFrame, 16), errs: make(chan error, 1)}
	go stream.read(bufio.NewReader(resp.Body))
	return stream
}

func (s *sseStream) read(reader *bufio.Reader) {
	var frame sseFrame
	var raw strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			s.errs <- err
			return
		}
		raw.WriteString(line)
		line = strings.TrimSuffix(line, "\n")
		switch {
		case line == "":
			frame.raw = raw.String()
			raw.Reset()
			if frame.event != "" {
				s.frames <- frame
			}
			frame = sseFrame{}
		case strings.HasPrefix(line, "id: "):
			frame.id, _ = strconv.ParseInt(strings.TrimPrefix(line, "id: "), 10, 64)
		case strings.HasPrefix(line, "event: "):
			frame.event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			frame.data = strings.TrimPrefix(line, "data: ")
		}
	}
}

func (s *sseStream) next(t *testing.T, event string) sseFrame {
	t.Helper()
	select {
	case frame := <-s.frames:
		if frame.event != event {
			t.Fatalf("next SSE frame is %q, want %q: %q", frame.event, event, frame.raw)
		}
		return frame
	case err := <-s.errs:
		t.Fatalf("SSE stream ended before %q: %v", event, err)
	case <-time.After(realtimeFrameTimeout):
		t.Fatalf("timed out waiting for SSE %q", event)
	}
	return sseFrame{}
}
