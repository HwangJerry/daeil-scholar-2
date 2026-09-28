// notification_inbox_service.go — Shapes published notices into the member's notification inbox
package service

import (
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

const (
	// Page sizes follow the feed, which lists the same notices.
	notificationPageDefaultSize = 10
	notificationPageMaxSize     = 20
	// noticeNotificationTitle is the fixed inbox title. Like the push sender
	// label it is a stable category name, not the admin-editable template title,
	// so the inbox reads the same no matter when a notice was sent.
	noticeNotificationTitle = "새 소식"
)

// NotificationInboxStore is the persistence the inbox needs. The inbox has no
// per-member rows: it is derived from notices, and only the newest notice SEQ
// the member has seen is stored.
type NotificationInboxStore interface {
	ListNoticeNotifications(beforeSeq, size int) ([]model.NoticeNotificationRow, error)
	GetLastSeenPostSeq(userSeq int) (int, error)
	CountUnseenNotices(userSeq int) (int, error)
	MarkSeenThrough(userSeq, lastSeenPostSeq int) error
}

type NotificationInboxService struct {
	store NotificationInboxStore
}

func NewNotificationInboxService(store NotificationInboxStore) *NotificationInboxService {
	return &NotificationInboxService{store: store}
}

// List returns one page of the member's inbox, newest first. beforeSeq <= 0
// reads the first page; nextCursor is set only when another page exists.
func (s *NotificationInboxService) List(userSeq, beforeSeq, size int) (*model.NotificationListResponse, error) {
	if size <= 0 {
		size = notificationPageDefaultSize
	}
	if size > notificationPageMaxSize {
		size = notificationPageMaxSize
	}
	lastSeen, err := s.store.GetLastSeenPostSeq(userSeq)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListNoticeNotifications(beforeSeq, size)
	if err != nil {
		return nil, err
	}
	hasMore := len(rows) > size
	if hasMore {
		rows = rows[:size]
	}
	items := make([]model.NotificationItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, noticeNotificationItem(row, lastSeen))
	}
	response := &model.NotificationListResponse{Items: items, HasMore: hasMore}
	if hasMore {
		cursor := "seq_" + strconv.Itoa(rows[len(rows)-1].SEQ)
		response.NextCursor = &cursor
	}
	return response, nil
}

// CountUnread is the bell's red-dot count.
func (s *NotificationInboxService) CountUnread(userSeq int) (int, error) {
	return s.store.CountUnseenNotices(userSeq)
}

// MarkSeenThrough clears the red dot up to the newest notice the app has shown.
// Reporting the SEQ the client actually displayed, rather than "now", keeps a
// notice published while the inbox was open unread.
func (s *NotificationInboxService) MarkSeenThrough(userSeq, lastSeenPostSeq int) error {
	return s.store.MarkSeenThrough(userSeq, lastSeenPostSeq)
}

// noticeNotificationItem mirrors the notice push: id matches its eventId and
// the deep link opens the same notice detail the push opens.
func noticeNotificationItem(row model.NoticeNotificationRow, lastSeenPostSeq int) model.NotificationItem {
	seq := strconv.Itoa(row.SEQ)
	return model.NotificationItem{
		ID:        "notice-" + seq,
		Type:      model.NotificationTypeAdminNotice,
		Title:     noticeNotificationTitle,
		Body:      row.Subject,
		CreatedAt: row.RegDate.UTC().Format(time.RFC3339),
		IsUnread:  row.SEQ > lastSeenPostSeq,
		DeepLink:  "/feed/" + seq,
		PostSeq:   row.SEQ,
	}
}
