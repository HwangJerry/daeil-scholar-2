package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

// noticeBroadcastStoreStub is a delivery store that can also list notice
// recipients. Pages are served in the order given, then exhausted, which is how
// the repository signals the end of the keyset walk.
type noticeBroadcastStoreStub struct {
	mu              sync.Mutex
	pages           [][]int
	pageCalls       []int
	pageErrors      []error
	devices         []model.PushDeliveryTarget
	noticePublished bool
	noticeChecks    int
}

func (s *noticeBroadcastStoreStub) GetPreferences(int) (*model.PushPreferences, error) {
	return nil, nil
}

func (s *noticeBroadcastStoreStub) ListDevices(int) ([]model.PushDeliveryTarget, error) {
	return s.devices, nil
}

func (s *noticeBroadcastStoreStub) DeleteDevice(string, string) error { return nil }

func (s *noticeBroadcastStoreStub) ListNoticeRecipientSeqs(afterUserSeq int, _ int) ([]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pageCalls = append(s.pageCalls, afterUserSeq)
	if len(s.pageErrors) > 0 {
		err := s.pageErrors[0]
		s.pageErrors = s.pageErrors[1:]
		if err != nil {
			return nil, err
		}
	}
	if len(s.pages) == 0 {
		return nil, nil
	}
	page := s.pages[0]
	s.pages = s.pages[1:]
	return page, nil
}

func (s *noticeBroadcastStoreStub) NoticeStillPublished(int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.noticeChecks++
	return s.noticePublished, nil
}

func (s *noticeBroadcastStoreStub) recordedPageCalls() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int(nil), s.pageCalls...)
}

// drainBroadcast collects everything the fan-out enqueued without starting the
// workers, so the queue contents can be asserted exactly.
func drainBroadcast(n *PushDeliveryNotifier) []pushDeliveryItem {
	items := make([]pushDeliveryItem, 0)
	for {
		select {
		case item := <-n.broadcast:
			items = append(items, item)
			continue
		default:
		}
		return items
	}
}

// waitForFanOut waits for the producer goroutine to finish enqueuing.
func waitForFanOut(t *testing.T, n *PushDeliveryNotifier) {
	t.Helper()
	done := make(chan struct{})
	go func() { n.producers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("notice fan-out did not finish")
	}
}

func TestNoticeFanOutEnqueuesOneItemPerRecipientAcrossPages(t *testing.T) {
	store := &noticeBroadcastStoreStub{pages: [][]int{{11, 12}, {13}}}
	notifier := NewPushDeliveryNotifier(store, &pushProviderStub{}, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(501, "장학금 안내")
	waitForFanOut(t, notifier)

	items := drainBroadcast(notifier)
	if len(items) != 3 {
		t.Fatalf("enqueued %d items, want one per recipient: %#v", len(items), items)
	}
	seen := map[int]int{}
	for _, item := range items {
		seen[item.recvrSeq]++
	}
	if seen[11] != 1 || seen[12] != 1 || seen[13] != 1 {
		t.Fatalf("per-recipient item counts = %#v", seen)
	}
	// The keyset cursor must advance past the highest member of each page, and
	// the walk must stop only on an empty page.
	calls := store.recordedPageCalls()
	if len(calls) != 3 || calls[0] != 0 || calls[1] != 12 || calls[2] != 13 {
		t.Fatalf("page cursors = %#v", calls)
	}
}

// Opted-out members are filtered by the recipient query, so a page that omits
// them must produce no items for them.
func TestNoticeFanOutSkipsRecipientsTheQueryExcluded(t *testing.T) {
	store := &noticeBroadcastStoreStub{pages: [][]int{{11}}}
	notifier := NewPushDeliveryNotifier(store, &pushProviderStub{}, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(501, "장학금 안내")
	waitForFanOut(t, notifier)

	items := drainBroadcast(notifier)
	if len(items) != 1 || items[0].recvrSeq != 11 {
		t.Fatalf("items = %#v, want only the opted-in recipient", items)
	}
}

// A blocking broadcast must not consume the chat shards, or every chat push
// sent during a fan-out would hit the non-blocking default and be dropped.
func TestNoticeBroadcastLeavesChatShardsFreeWhenItsOwnQueueIsFull(t *testing.T) {
	oversized := make([]int, 0, pushBroadcastCapacity*4)
	for i := 1; i <= cap(oversized); i++ {
		oversized = append(oversized, i)
	}
	store := &noticeBroadcastStoreStub{pages: [][]int{oversized}}
	notifier := NewPushDeliveryNotifier(store, &pushProviderStub{}, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(501, "장학금 안내")

	// Wait until the broadcast queue is saturated and the producer is parked.
	deadline := time.Now().Add(2 * time.Second)
	for len(notifier.broadcast) < pushBroadcastCapacity && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(notifier.broadcast) < pushBroadcastCapacity {
		t.Fatalf("broadcast queue holds %d, want it full at %d", len(notifier.broadcast), pushBroadcastCapacity)
	}

	notifier.NotifyMessageReceived(202, 101, "sender", &model.SendMessageResponse{MessageID: 9001}, "content")
	shard := notifier.shards[202%pushShardCount]
	if len(shard) != 1 {
		t.Fatalf("chat shard holds %d items, want the message enqueued despite the broadcast", len(shard))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := notifier.Stop(ctx); err != nil {
		t.Fatalf("Stop error = %v", err)
	}
}

// A transient page-query failure must not cost every recipient past the
// cursor; the keyset makes the retry read exactly the failed page.
func TestNoticeFanOutRetriesAFailedPageQuery(t *testing.T) {
	store := &noticeBroadcastStoreStub{
		pages:      [][]int{{11, 12}},
		pageErrors: []error{errors.New("transient"), nil},
	}
	notifier := NewPushDeliveryNotifier(store, &pushProviderStub{}, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(501, "장학금 안내")
	waitForFanOut(t, notifier)

	if len(drainBroadcast(notifier)) != 2 {
		t.Fatal("a retried page must still deliver its recipients")
	}
	calls := store.recordedPageCalls()
	if len(calls) < 2 || calls[0] != 0 || calls[1] != 0 {
		t.Fatalf("retry must re-read the same cursor: %#v", calls)
	}
}

// When every attempt fails the fan-out stops rather than looping, and says
// where it stopped.
func TestNoticeFanOutGivesUpAfterExhaustingPageRetries(t *testing.T) {
	store := &noticeBroadcastStoreStub{
		pages:      [][]int{{11}},
		pageErrors: []error{errors.New("down"), errors.New("down"), errors.New("down"), errors.New("down")},
	}
	notifier := NewPushDeliveryNotifier(store, &pushProviderStub{}, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(501, "장학금 안내")
	waitForFanOut(t, notifier)

	if len(drainBroadcast(notifier)) != 0 {
		t.Fatal("no recipient should be enqueued when every page read failed")
	}
	if calls := store.recordedPageCalls(); len(calls) != noticePageRetryAttempts {
		t.Fatalf("page attempts = %d, want %d", len(calls), noticePageRetryAttempts)
	}
}

func TestNoticePayloadCarriesRenderedTextAndFixedRoutingKeys(t *testing.T) {
	store := &noticeBroadcastStoreStub{
		pages:           [][]int{{11}},
		devices:         []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "a"}},
		noticePublished: true,
	}
	provider := &pushProviderStub{}
	notifier := NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(501, "장학금 안내")
	waitForFanOut(t, notifier)

	items := drainBroadcast(notifier)
	if len(items) != 1 {
		t.Fatalf("items = %#v", items)
	}
	notifier.deliver(context.Background(), items[0])
	if len(provider.payloads) != 1 {
		t.Fatalf("payloads = %#v", provider.payloads)
	}
	payload := provider.payloads[0]
	if payload.Type != "admin.notice" || payload.EventID != "notice-501" || payload.RecipientUserSeq != "11" ||
		payload.PostSeq != "501" || payload.Subject != "장학금 안내" {
		t.Fatalf("routing keys = %#v", payload)
	}
	// SenderName is a fixed label, never the editable template title, and
	// Preview is the raw subject rather than template output.
	if payload.SenderName != "새 소식" || payload.Preview != "장학금 안내" {
		t.Fatalf("senderName/preview = %q/%q", payload.SenderName, payload.Preview)
	}
	if payload.Title != "새 소식" || payload.Body != "장학금 안내" {
		t.Fatalf("rendered title/body = %q/%q", payload.Title, payload.Body)
	}
	if payload.TemplateKey != model.NotificationTemplateNoticeNew {
		t.Fatalf("template key = %q", payload.TemplateKey)
	}
	// A notice carries no chat identifiers.
	if payload.MessageID != "" || payload.ConversationUserSeq != "" || payload.SenderUserSeq != "" {
		t.Fatalf("message fields leaked into a notice payload: %#v", payload)
	}
}

func TestNoticeDeliverySkipsDeletedNotice(t *testing.T) {
	store := &noticeBroadcastStoreStub{
		pages:           [][]int{{11}},
		devices:         []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "a"}},
		noticePublished: false,
	}
	provider := &pushProviderStub{}
	notifier := NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(501, "장학금 안내")
	waitForFanOut(t, notifier)

	notifier.deliver(context.Background(), drainBroadcast(notifier)[0])
	if len(provider.calls) != 0 {
		t.Fatalf("a deleted notice was pushed to %d devices", len(provider.calls))
	}
	if store.noticeChecks == 0 {
		t.Fatal("publication guard was never consulted")
	}
}

// The guard answers per item, not per device: a recipient with several devices
// must cost exactly one check.
func TestNoticeDeliveryChecksPublicationOncePerRecipient(t *testing.T) {
	store := &noticeBroadcastStoreStub{
		pages: [][]int{{11}},
		devices: []model.PushDeliveryTarget{
			{Platform: "android", DeviceToken: "a"},
			{Platform: "ios", DeviceToken: "b", APNSEnvironment: "sandbox", BundleID: "com.daeil.dflhsafv2"},
		},
		noticePublished: true,
	}
	provider := &pushProviderStub{}
	notifier := NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(501, "장학금 안내")
	waitForFanOut(t, notifier)

	notifier.deliver(context.Background(), drainBroadcast(notifier)[0])
	if len(provider.calls) != 2 {
		t.Fatalf("provider calls = %d, want both devices", len(provider.calls))
	}
	if store.noticeChecks != 1 {
		t.Fatalf("publication checks = %d, want exactly one per item", store.noticeChecks)
	}
}

// A notice is an announcement, not chat: turning chat pushes off must not
// suppress it.
func TestNoticeDeliveryIgnoresMessagePreferences(t *testing.T) {
	store := &noticeMessageDisabledStore{noticeBroadcastStoreStub: noticeBroadcastStoreStub{
		pages:           [][]int{{11}},
		devices:         []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "a"}},
		noticePublished: true,
	}}
	provider := &pushProviderStub{}
	notifier := NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(501, "장학금 안내")
	waitForFanOut(t, notifier)

	notifier.deliver(context.Background(), drainBroadcast(notifier)[0])
	if len(provider.calls) != 1 {
		t.Fatalf("provider calls = %d, want the notice delivered anyway", len(provider.calls))
	}
}

type noticeMessageDisabledStore struct {
	noticeBroadcastStoreStub
}

func (s *noticeMessageDisabledStore) GetPreferences(int) (*model.PushPreferences, error) {
	return &model.PushPreferences{MessageEnabled: false, MessagePreviewEnabled: false, NoticeEnabled: true}, nil
}

func TestNoticeFanOutIgnoresNonPositiveNoticeSeq(t *testing.T) {
	store := &noticeBroadcastStoreStub{}
	notifier := NewPushDeliveryNotifier(store, &pushProviderStub{}, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(0, "장학금 안내")
	waitForFanOut(t, notifier)

	if len(store.recordedPageCalls()) != 0 || len(drainBroadcast(notifier)) != 0 {
		t.Fatal("a non-positive notice seq must not start a fan-out")
	}
}

// A store that cannot list recipients simply does not broadcast; it must not
// panic or block the caller.
func TestNoticeFanOutSkippedWhenStoreCannotListRecipients(t *testing.T) {
	notifier := NewPushDeliveryNotifier(&pushDeliveryStoreStub{}, &pushProviderStub{}, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(501, "장학금 안내")
	waitForFanOut(t, notifier)

	if len(drainBroadcast(notifier)) != 0 {
		t.Fatal("a store without recipient listing must enqueue nothing")
	}
}

// Shutdown must not be held open by an oversized broadcast, and the fan-out
// must never send on a closed queue.
func TestNoticeFanOutAbandonsRemainingRecipientsOnShutdown(t *testing.T) {
	oversized := make([]int, 0, pushBroadcastCapacity*4)
	for i := 1; i <= cap(oversized); i++ {
		oversized = append(oversized, i)
	}
	store := &noticeBroadcastStoreStub{pages: [][]int{oversized}}
	notifier := NewPushDeliveryNotifier(store, &pushProviderStub{}, NewTestNotificationTemplateService(nil), zerolog.Nop())
	notifier.NotifyNoticePublished(501, "장학금 안내")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := notifier.Stop(ctx); err != nil {
		t.Fatalf("Stop error = %v", err)
	}
}

// A notice published while the server is shutting down must be declined
// outright. Registering a producer after Stop began waiting would both misuse
// the WaitGroup and risk a send on an already-closed queue.
func TestNoticePublishedDuringShutdownIsDeclined(t *testing.T) {
	store := &noticeBroadcastStoreStub{pages: [][]int{{11, 12, 13}}}
	notifier := NewPushDeliveryNotifier(store, &pushProviderStub{}, NewTestNotificationTemplateService(nil), zerolog.Nop())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := notifier.Stop(ctx); err != nil {
		t.Fatalf("Stop error = %v", err)
	}

	// The queues are closed now; this must be a no-op rather than a panic.
	notifier.NotifyNoticePublished(501, "장학금 안내")
	notifier.producers.Wait()
	if calls := store.recordedPageCalls(); len(calls) != 0 {
		t.Fatalf("a fan-out started after shutdown: %#v", calls)
	}
}

// The same guarantee under contention: publishing concurrently with Stop must
// never panic, whichever order the two take.
func TestNoticePublishRacingShutdownIsSafe(t *testing.T) {
	store := &noticeBroadcastStoreStub{pages: [][]int{{11, 12, 13}}}
	notifier := NewPushDeliveryNotifier(store, &pushProviderStub{}, NewTestNotificationTemplateService(nil), zerolog.Nop())

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			notifier.NotifyNoticePublished(501, "장학금 안내")
		}()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := notifier.Stop(ctx); err != nil {
		t.Fatalf("Stop error = %v", err)
	}
	wg.Wait()
}
