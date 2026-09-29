package service

import (
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/realtime"
	"github.com/rs/zerolog"
)

func TestRealtimeNoticeNotifierPublishesNotificationCreatedToEveryStream(t *testing.T) {
	hub := realtime.NewHub(zerolog.Nop())
	first := hub.Subscribe(42)
	defer hub.Unsubscribe(first)
	second := hub.Subscribe(43)
	defer hub.Unsubscribe(second)

	NewRealtimeNoticeNotifier(hub).NotifyNoticePublished(501, "장학금 안내")

	for _, sub := range []*realtime.Subscriber{first, second} {
		select {
		case event := <-sub.Ch:
			payload := event.Payload.(map[string]any)
			if event.Type != "notification.created" || payload["notificationId"] != "notice-501" ||
				payload["postSeq"] != 501 || payload["type"] != "admin.notice" || payload["title"] != "새 소식" ||
				payload["body"] != "장학금 안내" || payload["deepLink"] != "/feed/501" {
				t.Fatalf("unexpected event %#v", event)
			}
			if _, err := time.Parse(time.RFC3339, payload["createdAt"].(string)); err != nil {
				t.Fatalf("createdAt must be RFC3339: %v", payload["createdAt"])
			}
			if id, ok := payload["eventId"].(int64); !ok || id <= 0 {
				t.Fatalf("hub must assign a numeric eventId: %#v", payload["eventId"])
			}
		case <-time.After(time.Second):
			t.Fatal("stream did not receive notification.created")
		}
	}
}

func TestRealtimeNoticeNotifierIgnoresInvalidSeq(t *testing.T) {
	hub := realtime.NewHub(zerolog.Nop())
	sub := hub.Subscribe(42)
	defer hub.Unsubscribe(sub)

	NewRealtimeNoticeNotifier(hub).NotifyNoticePublished(0, "무시")

	select {
	case event := <-sub.Ch:
		t.Fatalf("unexpected event %#v", event)
	case <-time.After(20 * time.Millisecond):
	}
}

type noticeNotifierRecorder struct {
	calls []int
}

func (r *noticeNotifierRecorder) NotifyNoticePublished(noticeSeq int, _ string) {
	r.calls = append(r.calls, noticeSeq)
}

func TestCompositeNoticeNotifierCallsEveryNonNilNotifier(t *testing.T) {
	first, second := &noticeNotifierRecorder{}, &noticeNotifierRecorder{}
	NewCompositeNoticeNotifier(first, nil, second).NotifyNoticePublished(7, "공지")

	if len(first.calls) != 1 || first.calls[0] != 7 || len(second.calls) != 1 || second.calls[0] != 7 {
		t.Fatalf("calls = %v, %v", first.calls, second.calls)
	}
}
