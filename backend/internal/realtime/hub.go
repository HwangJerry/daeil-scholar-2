// hub.go — In-process pub/sub hub fanning realtime events to per-user subscribers
package realtime

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const (
	eventHistoryLimit = 256
	subscriberBuffer  = eventHistoryLimit
)

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
	history     map[int][]Event
	nextEventID int64
	logger      zerolog.Logger
}

func NewHub(logger zerolog.Logger) *Hub {
	return &Hub{
		subscribers: make(map[int]map[*Subscriber]struct{}),
		history:     make(map[int][]Event),
		nextEventID: time.Now().UTC().UnixMilli(),
		logger:      logger,
	}
}

func (h *Hub) Subscribe(userSeq int, afterEventID ...int64) *Subscriber {
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
	if len(afterEventID) > 0 {
		for _, event := range h.history[userSeq] {
			if eventID(event) > afterEventID[0] {
				sub.Ch <- event
			}
		}
	}
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
	h.mu.Lock()
	h.nextEventID++
	if payload, ok := ev.Payload.(map[string]any); ok {
		payload["eventId"] = h.nextEventID
	}
	history := append(h.history[userSeq], ev)
	if len(history) > eventHistoryLimit {
		history = history[len(history)-eventHistoryLimit:]
	}
	h.history[userSeq] = history
	set, ok := h.subscribers[userSeq]
	if !ok {
		h.mu.Unlock()
		return
	}
	for sub := range set {
		select {
		case sub.Ch <- ev:
		default:
			h.logger.Warn().Int("userSeq", userSeq).Str("eventType", ev.Type).Msg("realtime subscriber buffer full, dropping event")
		}
	}
	h.mu.Unlock()
}

// PublishToConnected delivers an event to every user with an open stream right
// now, as one Publish per user. Each user gets a shallow copy of a map payload
// so each copy carries that user's own eventId. Users who are offline get
// nothing and nothing is kept for them: the REST endpoint is their catch-up.
// It returns how many users were published to.
func (h *Hub) PublishToConnected(ev Event) int {
	h.mu.RLock()
	userSeqs := make([]int, 0, len(h.subscribers))
	for userSeq := range h.subscribers {
		userSeqs = append(userSeqs, userSeq)
	}
	h.mu.RUnlock()
	for _, userSeq := range userSeqs {
		h.Publish(userSeq, Event{Type: ev.Type, Payload: copyPayload(ev.Payload)})
	}
	return len(userSeqs)
}

func copyPayload(payload interface{}) interface{} {
	original, ok := payload.(map[string]any)
	if !ok {
		return payload
	}
	copied := make(map[string]any, len(original)+1)
	for key, value := range original {
		copied[key] = value
	}
	return copied
}

func eventID(event Event) int64 {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return 0
	}
	id, _ := payload["eventId"].(int64)
	return id
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
