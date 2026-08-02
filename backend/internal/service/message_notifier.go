// message_notifier.go — Adapter that decouples MessageService from the realtime transport
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/realtime"
)

// MessageNotifier delivers message lifecycle events to interested transports
// (SSE, WebSocket, push, ...). MessageService depends only on this interface
// so the messaging domain stays independent from any specific delivery channel.
type MessageNotifier interface {
	NotifyMessageReceived(recvrSeq, senderSeq int, senderName string, accepted *model.SendMessageResponse, content string)
	NotifyMessageSent(senderSeq, recvrSeq int)
	NotifyMessagesRead(senderSeq, readerSeq int, throughMessageID int64, readAt string)
}

type nopMessageNotifier struct{}

func (nopMessageNotifier) NotifyMessageReceived(int, int, string, *model.SendMessageResponse, string) {
}
func (nopMessageNotifier) NotifyMessageSent(int, int)                 {}
func (nopMessageNotifier) NotifyMessagesRead(int, int, int64, string) {}

// NopMessageNotifier returns a notifier that drops every event. Useful for tests.
func NopMessageNotifier() MessageNotifier { return nopMessageNotifier{} }

type CompositeMessageNotifier struct {
	notifiers []MessageNotifier
}

func NewCompositeMessageNotifier(notifiers ...MessageNotifier) *CompositeMessageNotifier {
	return &CompositeMessageNotifier{notifiers: notifiers}
}

func (n *CompositeMessageNotifier) NotifyMessageReceived(recvrSeq, senderSeq int, senderName string, accepted *model.SendMessageResponse, content string) {
	for _, notifier := range n.notifiers {
		notifier.NotifyMessageReceived(recvrSeq, senderSeq, senderName, accepted, content)
	}
}

func (n *CompositeMessageNotifier) NotifyMessageSent(senderSeq, recvrSeq int) {
	for _, notifier := range n.notifiers {
		notifier.NotifyMessageSent(senderSeq, recvrSeq)
	}
}

func (n *CompositeMessageNotifier) NotifyMessagesRead(senderSeq, readerSeq int, throughMessageID int64, readAt string) {
	for _, notifier := range n.notifiers {
		notifier.NotifyMessagesRead(senderSeq, readerSeq, throughMessageID, readAt)
	}
}

// RealtimeMessageNotifier publishes message events to the in-process realtime hub.
type RealtimeMessageNotifier struct {
	hub *realtime.Hub
}

func NewRealtimeMessageNotifier(hub *realtime.Hub) *RealtimeMessageNotifier {
	return &RealtimeMessageNotifier{hub: hub}
}

func (n *RealtimeMessageNotifier) NotifyMessageReceived(recvrSeq, senderSeq int, senderName string, accepted *model.SendMessageResponse, content string) {
	n.hub.Publish(recvrSeq, realtime.Event{
		Type: "message.created",
		Payload: map[string]any{
			"messageId":           accepted.MessageID,
			"conversationUserSeq": senderSeq,
			"sender": map[string]any{
				"userSeq": senderSeq,
				"name":    senderName,
			},
			"preview":   content,
			"createdAt": accepted.CreatedAt,
		},
	})
}

func (n *RealtimeMessageNotifier) NotifyMessageSent(senderSeq, recvrSeq int) {
	n.hub.Publish(senderSeq, realtime.Event{
		Type: "conversation.updated",
		Payload: map[string]any{
			"conversationUserSeq": recvrSeq,
		},
	})
	n.hub.Publish(recvrSeq, realtime.Event{
		Type: "conversation.updated",
		Payload: map[string]any{
			"conversationUserSeq": senderSeq,
		},
	})
}

func (n *RealtimeMessageNotifier) NotifyMessagesRead(senderSeq, readerSeq int, throughMessageID int64, readAt string) {
	n.hub.Publish(senderSeq, realtime.Event{
		Type: "message.read",
		Payload: map[string]any{
			"conversationUserSeq": readerSeq,
			"throughMessageId":    throughMessageID,
			"readAt":              readAt,
		},
	})
}
