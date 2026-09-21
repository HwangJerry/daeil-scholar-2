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

// pushDeliveryItem is one queued notification. A non-empty verificationStatus
// marks a review notification; everything else is a message.
type pushDeliveryItem struct {
	verificationStatus model.VerificationStatus
	recvrSeq           int
	senderSeq          int
	senderName         string
	accepted           *model.SendMessageResponse
	content            string
}

func (i pushDeliveryItem) isVerificationReview() bool {
	return i.verificationStatus != ""
}

type PushDeliveryNotifier struct {
	store     PushDeliveryStore
	provider  PushProvider
	templates notificationTextRenderer
	logger    zerolog.Logger
	shards    [pushShardCount]chan pushDeliveryItem
	wg        sync.WaitGroup
}

// NewPushDeliveryNotifier creates the fan-out notifier. The renderer supplies
// the visible title and body of every push, which administrators may edit.
func NewPushDeliveryNotifier(
	store PushDeliveryStore,
	provider PushProvider,
	templates notificationTextRenderer,
	logger zerolog.Logger,
) *PushDeliveryNotifier {
	n := &PushDeliveryNotifier{store: store, provider: provider, templates: templates, logger: logger}
	for i := range n.shards {
		n.shards[i] = make(chan pushDeliveryItem, pushShardCapacity)
	}
	return n
}

func (n *PushDeliveryNotifier) Start() {
	for i := range n.shards {
		n.wg.Add(1)
		go func(shard <-chan pushDeliveryItem) {
			defer n.wg.Done()
			for item := range shard {
				n.deliver(context.Background(), item)
			}
		}(n.shards[i])
	}
}

func (n *PushDeliveryNotifier) Stop(ctx context.Context) error {
	for i := range n.shards {
		close(n.shards[i])
	}
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

func (n *PushDeliveryNotifier) deliver(ctx context.Context, item pushDeliveryItem) {
	var payload model.PushMessagePayload
	if item.isVerificationReview() {
		payload = n.buildVerificationPayload(item)
	} else {
		preferences, err := n.store.GetPreferences(item.recvrSeq)
		if err != nil {
			n.logger.Error().Int("recipient_seq", item.recvrSeq).Msg("push preferences lookup failed")
			return
		}
		if preferences != nil && !preferences.MessageEnabled {
			return
		}
		payload = n.buildMessagePayload(item, preferences)
	}
	targets, err := n.store.ListDevices(item.recvrSeq)
	if err != nil {
		n.logger.Error().Int("recipient_seq", item.recvrSeq).Msg("push device lookup failed")
		return
	}
	for _, target := range targets {
		for attempt := 0; attempt < 3; attempt++ {
			if guard, ok := n.store.(interface {
				MessageStillAvailable(int, int, int64) (bool, error)
			}); ok && !item.isVerificationReview() {
				available, checkErr := guard.MessageStillAvailable(item.senderSeq, item.recvrSeq, item.accepted.MessageID)
				if checkErr != nil || !available {
					return
				}
			}
			if guard, ok := n.store.(interface {
				VerificationStillCurrent(int, model.VerificationStatus) (bool, error)
			}); ok && item.isVerificationReview() {
				available, checkErr := guard.VerificationStillCurrent(item.recvrSeq, payload.VerificationStatus)
				if checkErr != nil || !available {
					return
				}
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
