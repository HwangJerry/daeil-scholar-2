// notice_published_notifier.go — Realtime inbox event and fan-out for a newly published notice
package service

import (
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/realtime"
)

// NotificationCreatedEvent tells open apps that the inbox gained an entry, so
// the bell can show its red dot without waiting for the next badge poll.
const NotificationCreatedEvent = "notification.created"

// RealtimeNoticeNotifier publishes a new notice to every open SSE stream. It
// does not depend on push delivery being enabled or on a member's push opt-out:
// the inbox lists every notice regardless of either.
type RealtimeNoticeNotifier struct {
	hub *realtime.Hub
}

func NewRealtimeNoticeNotifier(hub *realtime.Hub) *RealtimeNoticeNotifier {
	return &RealtimeNoticeNotifier{hub: hub}
}

// NotifyNoticePublished carries the same identifiers as the inbox item, so an
// app can insert or refetch it. The hub assigns each stream's numeric eventId.
func (n *RealtimeNoticeNotifier) NotifyNoticePublished(noticeSeq int, subject string) {
	if noticeSeq <= 0 {
		return
	}
	seq := strconv.Itoa(noticeSeq)
	n.hub.PublishToConnected(realtime.Event{
		Type: NotificationCreatedEvent,
		Payload: map[string]any{
			"notificationId": "notice-" + seq,
			"type":           model.NotificationTypeAdminNotice,
			"title":          noticeNotificationTitle,
			"body":           subject,
			"deepLink":       "/feed/" + seq,
			"postSeq":        noticeSeq,
			"createdAt":      time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// CompositeNoticeNotifier fans a published notice out to several transports,
// in order: the realtime inbox event always, the push broadcast when enabled.
type CompositeNoticeNotifier struct {
	notifiers []NoticePublishedNotifier
}

// NewCompositeNoticeNotifier skips nil notifiers, so an optional transport can
// be passed as-is. A typed nil pointer must be filtered by the caller.
func NewCompositeNoticeNotifier(notifiers ...NoticePublishedNotifier) *CompositeNoticeNotifier {
	kept := make([]NoticePublishedNotifier, 0, len(notifiers))
	for _, notifier := range notifiers {
		if notifier != nil {
			kept = append(kept, notifier)
		}
	}
	return &CompositeNoticeNotifier{notifiers: kept}
}

func (n *CompositeNoticeNotifier) NotifyNoticePublished(noticeSeq int, subject string) {
	for _, notifier := range n.notifiers {
		notifier.NotifyNoticePublished(noticeSeq, subject)
	}
}
