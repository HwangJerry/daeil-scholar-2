// notification_inbox_repo.go — Inbox reads over published notices plus the per-member last-seen marker
package repository

import (
	"database/sql"
	"errors"

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

// GetLastSeenPostSeq returns the newest notice SEQ the member has seen, or 0
// if the member never opened the inbox.
func (r *NotificationInboxRepository) GetLastSeenPostSeq(userSeq int) (int, error) {
	var lastSeen int
	err := r.db.Get(&lastSeen, `SELECT LAST_SEEN_POST_SEQ FROM ALUMNI_NOTIFICATION_INBOX_STATE WHERE USR_SEQ = ?`, userSeq)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return lastSeen, err
}

// CountUnseenNotices counts inbox notices newer than the member's last seen
// SEQ. A member who never opened the inbox has every inbox notice unseen.
func (r *NotificationInboxRepository) CountUnseenNotices(userSeq int) (int, error) {
	var count int
	err := r.db.Get(&count, `
		SELECT COUNT(*) FROM WEO_BOARDBBS
		WHERE `+noticeInboxFilter+`
		  AND SEQ > COALESCE(
		      (SELECT LAST_SEEN_POST_SEQ FROM ALUMNI_NOTIFICATION_INBOX_STATE WHERE USR_SEQ = ?),
		      0)
	`, NotificationInboxWindowDays, userSeq)
	return count, err
}

// MarkSeenThrough records that the member's app has shown every notice up to
// lastSeenPostSeq. The marker only moves forward (GREATEST), so an older or
// retried request cannot bring the red dot back, and it is clamped to the
// newest existing notice SEQ, so a bogus large value cannot hide notices that
// are published later. The clamp is read first rather than via INSERT ... SELECT,
// which would take shared locks on the notice rows it scans; a concurrent new
// notice can only make the clamp stale-low, which never hides it.
func (r *NotificationInboxRepository) MarkSeenThrough(userSeq, lastSeenPostSeq int) error {
	var newestNoticeSeq int
	if err := r.db.Get(&newestNoticeSeq, `SELECT COALESCE(MAX(SEQ), 0) FROM WEO_BOARDBBS WHERE GATE = 'NOTICE'`); err != nil {
		return err
	}
	if lastSeenPostSeq > newestNoticeSeq {
		lastSeenPostSeq = newestNoticeSeq
	}
	_, err := r.db.Exec(`
		INSERT INTO ALUMNI_NOTIFICATION_INBOX_STATE (USR_SEQ, LAST_SEEN_POST_SEQ, UPD_DATE)
		VALUES (?, ?, NOW())
		ON DUPLICATE KEY UPDATE
		    LAST_SEEN_POST_SEQ = GREATEST(LAST_SEEN_POST_SEQ, VALUES(LAST_SEEN_POST_SEQ)),
		    UPD_DATE = NOW()
	`, userSeq, lastSeenPostSeq)
	return err
}
