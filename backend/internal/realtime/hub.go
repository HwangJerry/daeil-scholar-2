// hub.go — In-process pub/sub hub fanning realtime events to per-user subscribers
package realtime

import (
	"sync"

	"github.com/rs/zerolog"
)

const subscriberBuffer = 8

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Subscriber struct {
	UserSeq int
	Ch      chan Event
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[int]map[*Subscriber]struct{}
	logger      zerolog.Logger
}

func NewHub(logger zerolog.Logger) *Hub {
	return &Hub{
		subscribers: make(map[int]map[*Subscriber]struct{}),
		logger:      logger,
	}
}

func (h *Hub) Subscribe(userSeq int) *Subscriber {
	sub := &Subscriber{
		UserSeq: userSeq,
		Ch:      make(chan Event, subscriberBuffer),
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.subscribers[userSeq]
	if !ok {
		set = make(map[*Subscriber]struct{})
		h.subscribers[userSeq] = set
	}
	set[sub] = struct{}{}
	return sub
}

func (h *Hub) Unsubscribe(s *Subscriber) {
	if s == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.subscribers[s.UserSeq]
	if !ok {
		return
	}
	delete(set, s)
	if len(set) == 0 {
		delete(h.subscribers, s.UserSeq)
	}
	close(s.Ch)
}

// Publish delivers an event to every active subscriber for userSeq. Non-blocking:
// if a subscriber's buffer is full the event is dropped (REST endpoints remain
// the source of truth, so a missed push only delays a refetch).
func (h *Hub) Publish(userSeq int, ev Event) {
	h.mu.RLock()
	set, ok := h.subscribers[userSeq]
	if !ok {
		h.mu.RUnlock()
		return
	}
	targets := make([]*Subscriber, 0, len(set))
	for sub := range set {
		targets = append(targets, sub)
	}
	h.mu.RUnlock()

	for _, sub := range targets {
		select {
		case sub.Ch <- ev:
		default:
			h.logger.Warn().Int("userSeq", userSeq).Str("eventType", ev.Type).Msg("realtime subscriber buffer full, dropping event")
		}
	}
}

func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, set := range h.subscribers {
		total += len(set)
	}
	return total
}
