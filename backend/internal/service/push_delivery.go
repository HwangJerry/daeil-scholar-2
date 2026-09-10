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
)

type PushDeliveryStore interface {
	GetPreferences(usrSeq int) (*model.PushPreferences, error)
	ListDevices(usrSeq int) ([]model.PushDeliveryTarget, error)
	DeleteDevice(platform, deviceToken string) error
}

type PushProvider interface {
	Send(ctx context.Context, target model.PushDeliveryTarget, payload model.PushMessagePayload) error
}

type pushDeliveryItem struct {
	reviewPayload *model.PushMessagePayload
	recvrSeq      int
	senderSeq     int
	senderName    string
	accepted      *model.SendMessageResponse
	content       string
}

type PushDeliveryNotifier struct {
	store    PushDeliveryStore
	provider PushProvider
	logger   zerolog.Logger
	shards   [pushShardCount]chan pushDeliveryItem
	wg       sync.WaitGroup
}

func NewPushDeliveryNotifier(store PushDeliveryStore, provider PushProvider, logger zerolog.Logger) *PushDeliveryNotifier {
	n := &PushDeliveryNotifier{store: store, provider: provider, logger: logger}
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
	now := time.Now().UTC()
	body := "동문 인증이 승인되었습니다. 이제 동문 커뮤니티를 이용할 수 있어요."
	if status == model.VerificationRejected {
		body = "동문 인증 신청이 반려되었습니다. 앱에서 사유를 확인하고 다시 신청해 주세요."
	}
	payload := &model.PushMessagePayload{Type: "verification.reviewed", EventID: "verification-" + strconv.Itoa(userSeq) + "-" + strconv.FormatInt(now.UnixNano(), 10), RecipientUserSeq: strconv.Itoa(userSeq), VerificationStatus: status, SenderName: "동문 인증 결과", Preview: body, CreatedAt: now.Format(time.RFC3339)}
	select {
	case n.shards[userSeq%pushShardCount] <- pushDeliveryItem{recvrSeq: userSeq, reviewPayload: payload}:
	default:
		n.logger.Warn().Int("recipient_seq", userSeq).Msg("verification push delivery queue full")
	}
}

func (n *PushDeliveryNotifier) NotifyMessageSent(int, int)                 {}
func (n *PushDeliveryNotifier) NotifyMessagesRead(int, int, int64, string) {}

func (n *PushDeliveryNotifier) deliver(ctx context.Context, item pushDeliveryItem) {
	var payload model.PushMessagePayload
	if item.reviewPayload != nil {
		payload = *item.reviewPayload
	} else {
		preferences, err := n.store.GetPreferences(item.recvrSeq)
		if err != nil {
			n.logger.Error().Int("recipient_seq", item.recvrSeq).Msg("push preferences lookup failed")
			return
		}
		if preferences != nil && !preferences.MessageEnabled {
			return
		}
		preview := item.content
		if preferences != nil && !preferences.MessagePreviewEnabled {
			preview = "새 메시지가 도착했습니다."
		}
		messageID := strconv.FormatInt(item.accepted.MessageID, 10)
		payload = model.PushMessagePayload{
			Type:                "message",
			EventID:             messageID,
			MessageID:           messageID,
			ConversationUserSeq: strconv.Itoa(item.senderSeq),
			SenderUserSeq:       strconv.Itoa(item.senderSeq),
			SenderName:          item.senderName,
			Preview:             preview,
			CreatedAt:           item.accepted.CreatedAt,
		}

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
			}); ok && item.reviewPayload == nil {
				available, checkErr := guard.MessageStillAvailable(item.senderSeq, item.recvrSeq, item.accepted.MessageID)
				if checkErr != nil || !available {
					return
				}
			}
			if guard, ok := n.store.(interface {
				VerificationStillCurrent(int, model.VerificationStatus) (bool, error)
			}); ok && item.reviewPayload != nil {
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
