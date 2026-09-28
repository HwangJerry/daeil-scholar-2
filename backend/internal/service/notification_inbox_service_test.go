package service

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

type notificationInboxStoreStub struct {
	rows      []model.NoticeNotificationRow
	lastSeen  *time.Time
	beforeSeq int
	size      int
	unseen    int
	marked    []int
	err       error
}

func (s *notificationInboxStoreStub) ListNoticeNotifications(beforeSeq, size int) ([]model.NoticeNotificationRow, error) {
	s.beforeSeq, s.size = beforeSeq, size
	if len(s.rows) > size+1 {
		return s.rows[:size+1], s.err
	}
	return s.rows, s.err
}

func (s *notificationInboxStoreStub) GetLastSeenAt(int) (*time.Time, error) { return s.lastSeen, s.err }

func (s *notificationInboxStoreStub) CountUnseenNotices(int) (int, error) { return s.unseen, s.err }

func (s *notificationInboxStoreStub) MarkSeen(userSeq int) error {
	s.marked = append(s.marked, userSeq)
	return s.err
}

var inboxSeoul = time.FixedZone("KST", 9*60*60)

func inboxRow(seq int, regDate time.Time) model.NoticeNotificationRow {
	return model.NoticeNotificationRow{SEQ: seq, Subject: "공지 " + strconv.Itoa(seq), RegDate: regDate}
}

func TestNotificationInboxShapesNoticesLikeThePush(t *testing.T) {
	regDate := time.Date(2026, 9, 28, 10, 0, 0, 0, inboxSeoul)
	store := &notificationInboxStoreStub{rows: []model.NoticeNotificationRow{{SEQ: 501, Subject: "장학금 안내", RegDate: regDate}}}
	response, err := NewNotificationInboxService(store).List(42, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := model.NotificationItem{
		ID: "notice-501", Type: "admin.notice", Title: "새 소식", Body: "장학금 안내",
		CreatedAt: "2026-09-28T01:00:00Z", IsUnread: true, DeepLink: "/feed/501", PostSeq: 501,
	}
	if len(response.Items) != 1 || response.Items[0] != want {
		t.Fatalf("items = %#v, want %#v", response.Items, want)
	}
	if response.HasMore || response.NextCursor != nil {
		t.Fatalf("single page must not advertise more: %#v", response)
	}
	if store.size != notificationPageDefaultSize || store.beforeSeq != 0 {
		t.Fatalf("store called with before %d size %d", store.beforeSeq, store.size)
	}
}

// Only notices registered after the last visit are unread; one registered at
// the exact moment of the visit was already on screen.
func TestNotificationInboxMarksOnlyNoticesAfterLastSeenUnread(t *testing.T) {
	lastSeen := time.Date(2026, 9, 28, 10, 0, 0, 0, inboxSeoul)
	store := &notificationInboxStoreStub{
		lastSeen: &lastSeen,
		rows: []model.NoticeNotificationRow{
			inboxRow(3, lastSeen.Add(time.Second)), inboxRow(2, lastSeen), inboxRow(1, lastSeen.Add(-time.Hour)),
		},
	}
	response, err := NewNotificationInboxService(store).List(42, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	got := []bool{response.Items[0].IsUnread, response.Items[1].IsUnread, response.Items[2].IsUnread}
	if got[0] != true || got[1] != false || got[2] != false {
		t.Fatalf("isUnread = %v, want [true false false]", got)
	}
}

func TestNotificationInboxPagesWithSeqCursorAndClampsSize(t *testing.T) {
	now := time.Now()
	rows := make([]model.NoticeNotificationRow, 0, 30)
	for seq := 130; seq > 100; seq-- {
		rows = append(rows, inboxRow(seq, now))
	}
	store := &notificationInboxStoreStub{rows: rows}
	response, err := NewNotificationInboxService(store).List(42, 131, 100)
	if err != nil {
		t.Fatal(err)
	}
	if store.size != notificationPageMaxSize || store.beforeSeq != 131 {
		t.Fatalf("store called with before %d size %d", store.beforeSeq, store.size)
	}
	if len(response.Items) != notificationPageMaxSize || !response.HasMore {
		t.Fatalf("items %d hasMore %v", len(response.Items), response.HasMore)
	}
	if response.NextCursor == nil || *response.NextCursor != "seq_111" {
		t.Fatalf("nextCursor = %v, want seq_111", response.NextCursor)
	}
}

func TestNotificationInboxReturnsEmptyItemsArray(t *testing.T) {
	response, err := NewNotificationInboxService(&notificationInboxStoreStub{}).List(42, 0, 10)
	if err != nil || response.Items == nil || len(response.Items) != 0 {
		t.Fatalf("response = %#v, err = %v", response, err)
	}
}

func TestNotificationInboxPropagatesStoreErrors(t *testing.T) {
	store := &notificationInboxStoreStub{err: errors.New("db down")}
	service := NewNotificationInboxService(store)
	if _, err := service.List(42, 0, 10); err == nil {
		t.Fatal("list must fail when the store fails")
	}
	if _, err := service.CountUnread(42); err == nil {
		t.Fatal("count must fail when the store fails")
	}
	if err := service.MarkAllSeen(42); err == nil || len(store.marked) != 1 || store.marked[0] != 42 {
		t.Fatalf("mark seen must reach the store for the member: %v %v", err, store.marked)
	}
}
