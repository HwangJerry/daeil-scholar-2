// push_notice_broadcast.go — Fan-out of a published notice (새 소식) to every
// registered device whose member has not opted out.
package service

import (
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

const (
	// noticeBroadcastSenderName labels notice pushes in the apps. There is no
	// human sender behind an announcement, and this is a routing/label key
	// rather than displayed text, so it stays fixed even when an administrator
	// edits the template title.
	noticeBroadcastSenderName = "새 소식"
	// noticeBroadcastPageSize is how many recipients one keyset page carries.
	// Small enough that a page never pins much memory on the 1GB server, large
	// enough that a few thousand members cost only a handful of queries.
	noticeBroadcastPageSize = 200
	// A page query that fails is retried rather than abandoning the rest of the
	// broadcast: the keyset cursor makes a retry read exactly the page that
	// failed, so retrying can neither duplicate nor skip a recipient. Waits
	// double from 200ms to 800ms across the attempts.
	noticePageRetryAttempts  = 4
	noticePageRetryBaseDelay = 200 * time.Millisecond
)

// NoticeRecipientLister is the recipient side of the broadcast. The delivery
// store satisfies it in production; a notifier whose store does not is simply
// unable to broadcast, which keeps the dependency optional.
type NoticeRecipientLister interface {
	ListNoticeRecipientSeqs(afterUserSeq int, limit int) ([]int, error)
}

// noticeBroadcast is the part of a fan-out that every recipient shares. The
// template is rendered once here rather than once per recipient, so an
// announcement to thousands of members costs one render.
type noticeBroadcast struct {
	seq       int
	subject   string
	createdAt string
	rendered  RenderedTemplate
}

// NotifyNoticePublished broadcasts a newly published notice. It returns as soon
// as the fan-out goroutine is started so the administrator's request is never
// held open behind the recipient pages.
//
// Known limitations, deliberately deferred:
//   - The broadcast is in-memory and at-most-once. A restart mid-fan-out loses
//     whatever was still queued; the unused ALUMNI_PUSH_OUTBOX table is the
//     natural home for a restart-safe version.
//   - GetPreferences selects NOTICE_ENABLED unconditionally, so migration 076
//     must be applied before this binary is deployed.
func (n *PushDeliveryNotifier) NotifyNoticePublished(noticeSeq int, subject string) {
	if noticeSeq <= 0 {
		return
	}
	lister, ok := n.store.(NoticeRecipientLister)
	if !ok {
		n.logger.Warn().Int("notice_seq", noticeSeq).Msg("notice push skipped: store cannot list recipients")
		return
	}
	broadcast := &noticeBroadcast{
		seq:       noticeSeq,
		subject:   subject,
		createdAt: time.Now().UTC().Format(time.RFC3339),
		rendered:  n.templates.Render(model.NotificationTemplateNoticeNew, map[string]string{"subject": subject}),
	}
	// Registering the producer under the shutdown lock is what makes Stop safe:
	// either this goroutine is counted before Stop begins waiting, or shutdown
	// already started and the broadcast is declined outright. Without the lock a
	// notice created during Stop could add to the WaitGroup after Wait, then
	// send on an already-closed queue.
	if !n.startProducer() {
		n.logger.Warn().Int("notice_seq", noticeSeq).Msg("notice push skipped: delivery is shutting down")
		return
	}
	go func() {
		defer n.producers.Done()
		n.fanOutNotice(lister, broadcast)
	}()
}

// fanOutNotice walks the recipient pages and enqueues exactly one item per
// opted-in member.
func (n *PushDeliveryNotifier) fanOutNotice(lister NoticeRecipientLister, broadcast *noticeBroadcast) {
	afterUserSeq, pages, recipients, dropped := 0, 0, 0, 0
	for {
		seqs, err := n.listNoticeRecipientPage(lister, afterUserSeq)
		if err != nil {
			n.logger.Error().
				Int("notice_seq", broadcast.seq).
				Int("after_user_seq", afterUserSeq).
				Int("recipients", recipients).
				Msg("notice push recipient lookup failed; fan-out stopped at this cursor")
			break
		}
		if len(seqs) == 0 {
			break
		}
		pages++
		for _, userSeq := range seqs {
			if userSeq > afterUserSeq {
				afterUserSeq = userSeq
			}
			if n.enqueueNotice(userSeq, broadcast) {
				recipients++
				continue
			}
			dropped++
		}
		if dropped > 0 {
			break
		}
	}
	n.logger.Info().
		Int("notice_seq", broadcast.seq).
		Int("recipients", recipients).
		Int("pages", pages).
		Int("dropped", dropped).
		Msg("notice push fan-out finished")
}

// listNoticeRecipientPage reads one page, retrying a failed read with doubling
// backoff. A transient database hiccup would otherwise cost every recipient
// past the cursor.
func (n *PushDeliveryNotifier) listNoticeRecipientPage(lister NoticeRecipientLister, afterUserSeq int) ([]int, error) {
	delay := noticePageRetryBaseDelay
	var err error
	for attempt := 0; attempt < noticePageRetryAttempts; attempt++ {
		var seqs []int
		seqs, err = lister.ListNoticeRecipientSeqs(afterUserSeq, noticeBroadcastPageSize)
		if err == nil {
			return seqs, nil
		}
		if attempt == noticePageRetryAttempts-1 {
			break
		}
		select {
		case <-time.After(delay):
		case <-n.closing:
			return nil, err
		}
		delay *= 2
	}
	return nil, err
}

// enqueueNotice blocks until the broadcast queue accepts the item. A broadcast
// is a single burst far larger than the queue's buffer, so the non-blocking
// enqueue that chat uses would silently drop most recipients. It waits on the
// broadcast queue rather than the recipient shards precisely so that a
// broadcast in progress cannot fill those shards and make chat and review
// pushes — which do drop on a full queue — miss their delivery.
//
// Shutdown is still prompt: a fan-out in progress abandons the rest of its
// recipients as soon as Stop signals.
func (n *PushDeliveryNotifier) enqueueNotice(userSeq int, broadcast *noticeBroadcast) bool {
	select {
	case n.broadcast <- pushDeliveryItem{recvrSeq: userSeq, notice: broadcast}:
		return true
	case <-n.closing:
		return false
	}
}

// buildNoticePayload assembles one recipient's notice push from the shared
// render. PostSeq and Subject are the routing key and the raw title the apps
// deep-link and list by; they never carry template output, exactly as
// SenderName and Preview do not for a message.
func (n *PushDeliveryNotifier) buildNoticePayload(item pushDeliveryItem) model.PushMessagePayload {
	broadcast := item.notice
	return model.PushMessagePayload{
		Type:             "admin.notice",
		EventID:          "notice-" + strconv.Itoa(broadcast.seq),
		RecipientUserSeq: strconv.Itoa(item.recvrSeq),
		PostSeq:          strconv.Itoa(broadcast.seq),
		Subject:          broadcast.subject,
		SenderName:       noticeBroadcastSenderName,
		Title:            broadcast.rendered.Title,
		Body:             broadcast.rendered.Body,
		Preview:          broadcast.subject,
		CreatedAt:        broadcast.createdAt,
		TemplateKey:      broadcast.rendered.Key,
		TemplateVersion:  broadcast.rendered.Version,
	}
}
