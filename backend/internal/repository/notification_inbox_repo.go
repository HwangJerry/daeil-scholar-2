// notification_inbox_repo.go — Inbox reads over published notices plus the per-member last-seen marker
package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// NotificationInboxWindowDays is how far back the inbox reaches. Older notices
// stay readable in the feed; they simply stop being "notifications".
const NotificationInboxWindowDays = 90

// noticeInboxFilter selects the notices the inbox shows: still published and
// registered inside the window. It is shared by the list and the unread count
// so the red dot can never disagree with the list.
const noticeInboxFilter = `GATE = 'NOTICE' AND OPEN_YN = 'Y'
		  AND REG_DATE >= NOW() - INTERVAL ? DAY`

type NotificationInboxRepository struct {
	db *sqlx.DB
}

func NewNotificationInboxRepository(db *sqlx.DB) *NotificationInboxRepository {
	return &NotificationInboxRepository{db: db}
}

// ListNoticeNotifications returns up to size+1 inbox notices, newest first, so
// the caller can tell whether another page exists. beforeSeq <= 0 reads the
// first page. The (GATE, OPEN_YN, SEQ) index serves the ordering directly.
func (r *NotificationInboxRepository) ListNoticeNotifications(beforeSeq, size int) ([]model.NoticeNotificationRow, error) {
	query := `SELECT SEQ, SUBJECT, REG_DATE FROM WEO_BOARDBBS WHERE ` + noticeInboxFilter
	args := []interface{}{NotificationInboxWindowDays}
	if beforeSeq > 0 {
		query += ` AND SEQ < ?`
		args = append(args, beforeSeq)
	}
	query += ` ORDER BY SEQ DESC LIMIT ?`
	args = append(args, size+1)

	rows := []model.NoticeNotificationRow{}
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

// GetLastSeenAt returns when the member last opened the inbox, or nil if never.
func (r *NotificationInboxRepository) GetLastSeenAt(userSeq int) (*time.Time, error) {
	var lastSeen time.Time
	err := r.db.Get(&lastSeen, `SELECT LAST_SEEN_AT FROM ALUMNI_NOTIFICATION_INBOX_STATE WHERE USR_SEQ = ?`, userSeq)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lastSeen, nil
}

// CountUnseenNotices counts inbox notices registered after the member last
// opened the inbox. A member who never opened it has every inbox notice unseen.
func (r *NotificationInboxRepository) CountUnseenNotices(userSeq int) (int, error) {
	var count int
	err := r.db.Get(&count, `
		SELECT COUNT(*) FROM WEO_BOARDBBS
		WHERE `+noticeInboxFilter+`
		  AND REG_DATE > COALESCE(
		      (SELECT LAST_SEEN_AT FROM ALUMNI_NOTIFICATION_INBOX_STATE WHERE USR_SEQ = ?),
		      '1970-01-01')
	`, NotificationInboxWindowDays, userSeq)
	return count, err
}

// MarkSeen records that the member opened the inbox now.
func (r *NotificationInboxRepository) MarkSeen(userSeq int) error {
	_, err := r.db.Exec(`
		INSERT INTO ALUMNI_NOTIFICATION_INBOX_STATE (USR_SEQ, LAST_SEEN_AT, UPD_DATE)
		VALUES (?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE LAST_SEEN_AT = NOW(), UPD_DATE = NOW()
	`, userSeq)
	return err
}
