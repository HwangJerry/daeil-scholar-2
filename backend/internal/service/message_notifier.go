// message_notifier.go — Adapter that decouples MessageService from the realtime transport
package service

import "github.com/dflh-saf/backend/internal/realtime"

// MessageNotifier delivers message lifecycle events to interested transports
// (SSE, WebSocket, push, ...). MessageService depends only on this interface
// so the messaging domain stays independent from any specific delivery channel.
type MessageNotifier interface {
	NotifyMessageReceived(recvrSeq, senderSeq int, senderName string)
	NotifyMessageSent(senderSeq, recvrSeq int)
}

type nopMessageNotifier struct{}

func (nopMessageNotifier) NotifyMessageReceived(int, int, string) {}
func (nopMessageNotifier) NotifyMessageSent(int, int)             {}

// NopMessageNotifier returns a notifier that drops every event. Useful for tests.
func NopMessageNotifier() MessageNotifier { return nopMessageNotifier{} }

// RealtimeMessageNotifier publishes message events to the in-process realtime hub.
type RealtimeMessageNotifier struct {
	hub *realtime.Hub
}

func NewRealtimeMessageNotifier(hub *realtime.Hub) *RealtimeMessageNotifier {
	return &RealtimeMessageNotifier{hub: hub}
}

func (n *RealtimeMessageNotifier) NotifyMessageReceived(recvrSeq, senderSeq int, senderName string) {
	n.hub.Publish(recvrSeq, realtime.Event{
		Type: "message.new",
		Payload: map[string]any{
			"fromSeq":  senderSeq,
			"fromName": senderName,
		},
	})
}

func (n *RealtimeMessageNotifier) NotifyMessageSent(senderSeq, recvrSeq int) {
	n.hub.Publish(senderSeq, realtime.Event{
		Type: "message.sent",
		Payload: map[string]any{
			"toSeq": recvrSeq,
		},
	})
}
