package service

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

var (
	ErrPushInvalidToken = errors.New("invalid push token")
	ErrPushTransient    = errors.New("transient push provider error")
)

const (
	pushShardCount    = 4
	pushShardCapacity = 256
	// A notice broadcast gets its own queue and workers. It enqueues blocking
	// and in one burst, so sharing the recipient shards would fill them and
	// make chat and verification pushes — which drop on a full queue — vanish
	// for the whole fan-out.
	pushBroadcastCapacity    = 256
	pushBroadcastWorkerCount = 2
	// verificationReviewSenderName labels review pushes in the apps. There is no
	// human sender behind a review, and this is a routing/label key rather than
	// displayed text, so it stays fixed even when the title is edited.
	verificationReviewSenderName = "동문 인증 결과"
)

type PushDeliveryStore interface {
	GetPreferences(usrSeq int) (*model.PushPreferences, error)
	ListDevices(usrSeq int) ([]model.PushDeliveryTarget, error)
	DeleteDevice(platform, deviceToken string) error
}

type PushProvider interface {
	Send(ctx context.Context, target model.PushDeliveryTarget, payload model.PushMessagePayload) error
}

// pushDeliveryItem is one queued notification. A non-nil notice marks a notice
// broadcast, a non-empty verificationStatus marks a review notification, and
// everything else is a message.
type pushDeliveryItem struct {
	verificationStatus model.VerificationStatus
	notice             *noticeBroadcast
	recvrSeq           int
	senderSeq          int
	senderName         string
	accepted           *model.SendMessageResponse
	content            string
}

func (i pushDeliveryItem) isVerificationReview() bool {
	return i.verificationStatus != ""
}

func (i pushDeliveryItem) isNoticeBroadcast() bool {
	return i.notice != nil
}

type PushDeliveryNotifier struct {
	store     PushDeliveryStore
	provider  PushProvider
	templates notificationTextRenderer
	logger    zerolog.Logger
	shards    [pushShardCount]chan pushDeliveryItem
	// broadcast carries notice fan-outs, kept apart from the recipient shards
	// so a burst of announcements cannot starve chat.
	broadcast chan pushDeliveryItem
	wg        sync.WaitGroup
	// closing is shut before the queues are, so a blocking broadcast producer
	// can abandon its fan-out instead of delaying shutdown or, worse, sending
	// on a closed channel. producers tracks those goroutines so Stop only
	// closes the queues once every one of them has returned, and shutdownMu
	// makes "register a producer" and "begin shutting down" mutually exclusive
	// so a producer can never be added after Stop started waiting.
	closing    chan struct{}
	shutdownMu sync.Mutex
	closed     bool
	producers  sync.WaitGroup
}

// startProducer registers a broadcast goroutine unless shutdown has begun. It
// reports whether the caller may proceed; a false result means the queues are
// about to close and the caller must not enqueue anything.
func (n *PushDeliveryNotifier) startProducer() bool {
	n.shutdownMu.Lock()
	defer n.shutdownMu.Unlock()
	if n.closed {
		return false
	}
	n.producers.Add(1)
	return true
}

// beginShutdown closes the shutdown signal exactly once and blocks any further
// producer from registering.
func (n *PushDeliveryNotifier) beginShutdown() {
	n.shutdownMu.Lock()
	defer n.shutdownMu.Unlock()
	if n.closed {
		return
	}
	n.closed = true
	close(n.closing)
}

// NewPushDeliveryNotifier creates the fan-out notifier. The renderer supplies
// the visible title and body of every push, which administrators may edit.
func NewPushDeliveryNotifier(
	store PushDeliveryStore,
	provider PushProvider,
	templates notificationTextRenderer,
	logger zerolog.Logger,
) *PushDeliveryNotifier {
	n := &PushDeliveryNotifier{
		store:     store,
		provider:  provider,
		templates: templates,
		logger:    logger,
		broadcast: make(chan pushDeliveryItem, pushBroadcastCapacity),
		closing:   make(chan struct{}),
	}
	for i := range n.shards {
		n.shards[i] = make(chan pushDeliveryItem, pushShardCapacity)
	}
	return n
}

func (n *PushDeliveryNotifier) Start() {
	consume := func(queue <-chan pushDeliveryItem) {
		defer n.wg.Done()
		for item := range queue {
			n.deliver(context.Background(), item)
		}
	}
	for i := range n.shards {
		n.wg.Add(1)
		go consume(n.shards[i])
	}
	// Broadcast workers run the same delivery code; only the queue differs.
	// Ordering across recipients is meaningless for an announcement, so more
	// than one worker is safe.
	for i := 0; i < pushBroadcastWorkerCount; i++ {
		n.wg.Add(1)
		go consume(n.broadcast)
	}
}

func (n *PushDeliveryNotifier) Stop(ctx context.Context) error {
	n.beginShutdown()
	producersDone := make(chan struct{})
	go func() { n.producers.Wait(); close(producersDone) }()
	select {
	case <-producersDone:
	case <-ctx.Done():
		return ctx.Err()
	}
	for i := range n.shards {
		close(n.shards[i])
	}
	close(n.broadcast)
	done := make(chan struct{})
	go func() { n.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (n *PushDeliveryNotifier) NotifyMessageReceived(recvrSeq, senderSeq int, senderName string, accepted *model.SendMessageResponse, content string) {
	item := pushDeliveryItem{recvrSeq: recvrSeq, senderSeq: senderSeq, senderName: senderName, accepted: accepted, content: content}
	select {
	case n.shards[recvrSeq%pushShardCount] <- item:
	default:
		n.logger.Warn().Int("recipient_seq", recvrSeq).Msg("push delivery queue full")
	}
}

// Review notifications are account-service events, independent of message preferences.
// Rejection details are fetched in the authenticated app, never exposed on a lock screen.
func (n *PushDeliveryNotifier) NotifyVerificationReviewed(userSeq int, status model.VerificationStatus) {
	if userSeq <= 0 || (status != model.VerificationApproved && status != model.VerificationRejected) {
		return
	}
	select {
	case n.shards[userSeq%pushShardCount] <- pushDeliveryItem{recvrSeq: userSeq, verificationStatus: status}:
	default:
		n.logger.Warn().Int("recipient_seq", userSeq).Msg("verification push delivery queue full")
	}
}

func (n *PushDeliveryNotifier) NotifyMessageSent(int, int)                 {}
func (n *PushDeliveryNotifier) NotifyMessagesRead(int, int, int64, string) {}

// buildVerificationPayload renders one review notification. Rejection details
// are fetched in the authenticated app, never exposed on a lock screen, so the
// body carries only the outcome.
func (n *PushDeliveryNotifier) buildVerificationPayload(item pushDeliveryItem) model.PushMessagePayload {
	templateKey := model.NotificationTemplateVerificationApproved
	if item.verificationStatus == model.VerificationRejected {
		templateKey = model.NotificationTemplateVerificationRejected
	}
	rendered := n.templates.Render(templateKey, nil)
	now := time.Now().UTC()
	return model.PushMessagePayload{
		Type:               "verification.reviewed",
		EventID:            "verification-" + strconv.Itoa(item.recvrSeq) + "-" + strconv.FormatInt(now.UnixNano(), 10),
		RecipientUserSeq:   strconv.Itoa(item.recvrSeq),
		VerificationStatus: item.verificationStatus,
		SenderName:         verificationReviewSenderName,
		Title:              rendered.Title,
		Body:               rendered.Body,
		// A review carries no message content, so the data key repeats the
		// displayed body rather than holding anything the body hides.
		Preview:         rendered.Body,
		CreatedAt:       now.Format(time.RFC3339),
		TemplateKey:     rendered.Key,
		TemplateVersion: rendered.Version,
	}
}

// buildMessagePayload renders one new-message notification. It renders once per
// recipient rather than once per device; the template service caches, so
// repeated renders cost a map lookup.
func (n *PushDeliveryNotifier) buildMessagePayload(
	item pushDeliveryItem,
	preferences *model.PushPreferences,
) model.PushMessagePayload {
	previewEnabled := preferences == nil || preferences.MessagePreviewEnabled

	var rendered RenderedTemplate
	// Preview holds the raw snippet the apps list conversations by. With
	// previews off it must not carry the content at all, so it repeats the
	// masked body instead.
	preview := item.content
	if previewEnabled {
		rendered = n.templates.Render(
			model.NotificationTemplateMessagePreviewOn,
			map[string]string{"senderName": item.senderName, "content": item.content},
		)
	} else {
		rendered = n.templates.Render(
			model.NotificationTemplateMessagePreviewOff,
			map[string]string{"senderName": item.senderName},
		)
		preview = rendered.Body
	}

	messageID := strconv.FormatInt(item.accepted.MessageID, 10)
	return model.PushMessagePayload{
		Type:                "message",
		EventID:             messageID,
		RecipientUserSeq:    strconv.Itoa(item.recvrSeq),
		MessageID:           messageID,
		ConversationUserSeq: strconv.Itoa(item.senderSeq),
		SenderUserSeq:       strconv.Itoa(item.senderSeq),
		// SenderName stays the real sender: the apps route and label
		// conversations by it, independently of the editable title.
		SenderName:      item.senderName,
		Title:           rendered.Title,
		Body:            rendered.Body,
		Preview:         preview,
		CreatedAt:       item.accepted.CreatedAt,
		TemplateKey:     rendered.Key,
		TemplateVersion: rendered.Version,
	}
}

// buildPayload renders the item. The second result is false when the item must
// not be sent at all, which today only a message preference can decide.
func (n *PushDeliveryNotifier) buildPayload(item pushDeliveryItem) (model.PushMessagePayload, bool) {
	switch {
	case item.isNoticeBroadcast():
		// A notice is an announcement, not chat: MessageEnabled governs
		// conversations only, and the NOTICE_ENABLED opt-out was already
		// applied by the recipient query, so no preference is read here.
		return n.buildNoticePayload(item), true
	case item.isVerificationReview():
		return n.buildVerificationPayload(item), true
	default:
		preferences, err := n.store.GetPreferences(item.recvrSeq)
		if err != nil {
			n.logger.Error().Int("recipient_seq", item.recvrSeq).Msg("push preferences lookup failed")
			return model.PushMessagePayload{}, false
		}
		if preferences != nil && !preferences.MessageEnabled {
			return model.PushMessagePayload{}, false
		}
		return n.buildMessagePayload(item, preferences), true
	}
}

// stillDeliverable re-checks, once per queued item, that the thing this push
// announces still exists. Items wait in a queue, so between enqueue and send an
// administrator can delete the notice or a review can be superseded. For these
// the answer cannot differ between a recipient's own devices, so asking once
// per item is both correct and cheapest. A store that does not implement the
// guard for that kind simply has nothing to re-check.
func (n *PushDeliveryNotifier) stillDeliverable(item pushDeliveryItem, payload model.PushMessagePayload) bool {
	switch {
	case item.isNoticeBroadcast():
		guard, ok := n.store.(interface {
			NoticeStillPublished(int) (bool, error)
		})
		if !ok {
			return true
		}
		published, err := guard.NoticeStillPublished(item.notice.seq)
		return err == nil && published
	case item.isVerificationReview():
		guard, ok := n.store.(interface {
			VerificationStillCurrent(int, model.VerificationStatus) (bool, error)
		})
		if !ok {
			return true
		}
		current, err := guard.VerificationStillCurrent(item.recvrSeq, payload.VerificationStatus)
		return err == nil && current
	default:
		// A message is guarded per delivery attempt instead; see
		// messageStillAvailable.
		return true
	}
}

// messageStillAvailable re-checks the erasure and privacy guard before every
// single send. Unlike the notice and review guards, whose answer cannot change
// between one recipient's own devices, this one protects erased content: a
// member who withdraws, or a message that is deleted, must not reach even the
// second device of the same recipient. A recipient has one or two devices, so
// asking per attempt costs almost nothing.
func (n *PushDeliveryNotifier) messageStillAvailable(item pushDeliveryItem) bool {
	if item.isNoticeBroadcast() || item.isVerificationReview() {
		return true
	}
	guard, ok := n.store.(interface {
		MessageStillAvailable(int, int, int64) (bool, error)
	})
	if !ok {
		return true
	}
	available, err := guard.MessageStillAvailable(item.senderSeq, item.recvrSeq, item.accepted.MessageID)
	return err == nil && available
}

func (n *PushDeliveryNotifier) deliver(ctx context.Context, item pushDeliveryItem) {
	payload, ok := n.buildPayload(item)
	if !ok {
		return
	}
	if !n.stillDeliverable(item, payload) {
		return
	}
	targets, err := n.store.ListDevices(item.recvrSeq)
	if err != nil {
		n.logger.Error().Int("recipient_seq", item.recvrSeq).Msg("push device lookup failed")
		return
	}
	for _, target := range targets {
		for attempt := 0; attempt < 3; attempt++ {
			if !n.messageStillAvailable(item) {
				return
			}
			err = n.provider.Send(ctx, target, payload)
			if err == nil {
				break
			}
			if errors.Is(err, ErrPushInvalidToken) {
				if deleteErr := n.store.DeleteDevice(target.Platform, target.DeviceToken); deleteErr != nil {
					n.logger.Error().Str("platform", target.Platform).Msg("invalid push device cleanup failed")
				}
				break
			}
			if !errors.Is(err, ErrPushTransient) {
				n.logger.Warn().Str("platform", target.Platform).Msg("push delivery rejected")
				break
			}
			if attempt == 2 {
				n.logger.Warn().Str("platform", target.Platform).Msg("push delivery retries exhausted")
			}
		}
	}
}
