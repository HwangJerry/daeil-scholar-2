// notification_repo.go — Database access layer for ALUMNI_NOTIFICATION table
package repository

import (
	"database/sql"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// NotificationRepository handles CRUD operations for ALUMNI_NOTIFICATION.
type NotificationRepository struct {
	DB *sqlx.DB
}

// NewNotificationRepository creates a new NotificationRepository.
func NewNotificationRepository(db *sqlx.DB) *NotificationRepository {
	return &NotificationRepository{DB: db}
}

// Insert creates a new notification record with REG_DATE set to NOW().
func (r *NotificationRepository) Insert(usrSeq int, notiType, title, body string, refSeq *int) error {
	_, err := r.DB.Exec(`
		INSERT INTO ALUMNI_NOTIFICATION (USR_SEQ, AN_TYPE, AN_TITLE, AN_BODY, AN_REF_SEQ, AN_READ_YN, REG_DATE)
		VALUES (?, ?, ?, ?, ?, 'N', NOW())
	`, usrSeq, notiType, title, body, refSeq)
	return err
}

// GetByUser returns paginated notifications for a user ordered by newest first,
// along with the total count of notifications for that user.
func (r *NotificationRepository) GetByUser(usrSeq, page, size int) ([]model.Notification, int, error) {
	offset := (page - 1) * size

	var total int
	err := r.DB.Get(&total, `
		SELECT COUNT(*) FROM ALUMNI_NOTIFICATION WHERE USR_SEQ = ?
	`, usrSeq)
	if err != nil {
		return nil, 0, err
	}

	var items []model.Notification
	err = r.DB.Select(&items, `
		SELECT AN_SEQ, USR_SEQ, AN_TYPE, AN_TITLE,
		       IFNULL(AN_BODY, '') AS AN_BODY,
		       AN_REF_SEQ, AN_READ_YN,
		       DATE_FORMAT(REG_DATE, '%Y-%m-%d %H:%i:%s') AS REG_DATE
		FROM ALUMNI_NOTIFICATION
		WHERE USR_SEQ = ?
		ORDER BY REG_DATE DESC
		LIMIT ? OFFSET ?
	`, usrSeq, size, offset)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// GetUnreadCount returns the number of unread notifications for a user.
func (r *NotificationRepository) GetUnreadCount(usrSeq int) (int, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM ALUMNI_NOTIFICATION
		WHERE USR_SEQ = ? AND AN_READ_YN = 'N'
	`, usrSeq)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// MarkAsRead marks a single notification as read for the given user.
func (r *NotificationRepository) MarkAsRead(anSeq, usrSeq int) error {
	result, err := r.DB.Exec(`
		UPDATE ALUMNI_NOTIFICATION
		SET AN_READ_YN = 'Y'
		WHERE AN_SEQ = ? AND USR_SEQ = ?
	`, anSeq, usrSeq)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// MarkAllAsRead marks all unread notifications as read for the given user.
func (r *NotificationRepository) MarkAllAsRead(usrSeq int) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_NOTIFICATION
		SET AN_READ_YN = 'Y'
		WHERE USR_SEQ = ? AND AN_READ_YN = 'N'
	`, usrSeq)
	return err
}

// DeleteOld removes notifications older than the specified number of days.
// Returns the number of deleted rows.
func (r *NotificationRepository) DeleteOld(days int) (int64, error) {
	result, err := r.DB.Exec(`
		DELETE FROM ALUMNI_NOTIFICATION
		WHERE REG_DATE < DATE_SUB(NOW(), INTERVAL ? DAY)
	`, days)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
