// message_repo.go — Database access layer for alumni direct messaging
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// MessageRepository handles ALUMNI_MESSAGE table queries.
type MessageRepository struct {
	DB *sqlx.DB
}

// NewMessageRepository creates a new MessageRepository.
func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{DB: db}
}

// InsertMessage creates a new message record.
func (r *MessageRepository) InsertMessage(senderSeq int, recvrSeq int, content string) error {
	_, err := r.DB.Exec(`
		INSERT INTO ALUMNI_MESSAGE (AM_SENDER_SEQ, AM_RECVR_SEQ, AM_CONTENT, AM_READ_YN, REG_DATE)
		VALUES (?, ?, ?, 'N', NOW())
	`, senderSeq, recvrSeq, content)
	return err
}

// GetInbox returns received messages for a user, paginated.
func (r *MessageRepository) GetInbox(usrSeq int, page int, size int) ([]model.Message, int, error) {
	var total int
	if err := r.DB.Get(&total, `
		SELECT COUNT(*) FROM ALUMNI_MESSAGE WHERE AM_RECVR_SEQ = ? AND AM_DEL_RECVR = 'N'
	`, usrSeq); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	var messages []model.Message
	if err := r.DB.Select(&messages, `
		SELECT am.AM_SEQ, am.AM_SENDER_SEQ, am.AM_RECVR_SEQ, am.AM_CONTENT,
			am.AM_READ_YN,
			IFNULL(DATE_FORMAT(am.REG_DATE, '%Y-%m-%d %H:%i:%s'), '') AS REG_DATE,
			IFNULL(DATE_FORMAT(am.READ_DATE, '%Y-%m-%d %H:%i:%s'), '') AS READ_DATE,
			IFNULL(s.USR_NAME, '') AS SENDER_NAME,
			IFNULL(rv.USR_NAME, '') AS RECVR_NAME
		FROM ALUMNI_MESSAGE am
		LEFT JOIN WEO_MEMBER s ON am.AM_SENDER_SEQ = s.USR_SEQ
		LEFT JOIN WEO_MEMBER rv ON am.AM_RECVR_SEQ = rv.USR_SEQ
		WHERE am.AM_RECVR_SEQ = ? AND am.AM_DEL_RECVR = 'N'
		ORDER BY am.REG_DATE DESC
		LIMIT ? OFFSET ?
	`, usrSeq, size, offset); err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// GetOutbox returns sent messages for a user, paginated.
func (r *MessageRepository) GetOutbox(usrSeq int, page int, size int) ([]model.Message, int, error) {
	var total int
	if err := r.DB.Get(&total, `
		SELECT COUNT(*) FROM ALUMNI_MESSAGE WHERE AM_SENDER_SEQ = ? AND AM_DEL_SENDER = 'N'
	`, usrSeq); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	var messages []model.Message
	if err := r.DB.Select(&messages, `
		SELECT am.AM_SEQ, am.AM_SENDER_SEQ, am.AM_RECVR_SEQ, am.AM_CONTENT,
			am.AM_READ_YN,
			IFNULL(DATE_FORMAT(am.REG_DATE, '%Y-%m-%d %H:%i:%s'), '') AS REG_DATE,
			IFNULL(DATE_FORMAT(am.READ_DATE, '%Y-%m-%d %H:%i:%s'), '') AS READ_DATE,
			IFNULL(s.USR_NAME, '') AS SENDER_NAME,
			IFNULL(rv.USR_NAME, '') AS RECVR_NAME
		FROM ALUMNI_MESSAGE am
		LEFT JOIN WEO_MEMBER s ON am.AM_SENDER_SEQ = s.USR_SEQ
		LEFT JOIN WEO_MEMBER rv ON am.AM_RECVR_SEQ = rv.USR_SEQ
		WHERE am.AM_SENDER_SEQ = ? AND am.AM_DEL_SENDER = 'N'
		ORDER BY am.REG_DATE DESC
		LIMIT ? OFFSET ?
	`, usrSeq, size, offset); err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// MarkAsRead marks a message as read.
func (r *MessageRepository) MarkAsRead(amSeq int, usrSeq int) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_MESSAGE SET AM_READ_YN = 'Y', READ_DATE = NOW()
		WHERE AM_SEQ = ? AND AM_RECVR_SEQ = ?
	`, amSeq, usrSeq)
	return err
}

// DeleteMessage soft-deletes a message for the requesting user.
func (r *MessageRepository) DeleteMessage(amSeq int, usrSeq int) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_MESSAGE
		SET AM_DEL_SENDER = CASE WHEN AM_SENDER_SEQ = ? THEN 'Y' ELSE AM_DEL_SENDER END,
			AM_DEL_RECVR  = CASE WHEN AM_RECVR_SEQ  = ? THEN 'Y' ELSE AM_DEL_RECVR  END
		WHERE AM_SEQ = ? AND (AM_SENDER_SEQ = ? OR AM_RECVR_SEQ = ?)
	`, usrSeq, usrSeq, amSeq, usrSeq, usrSeq)
	return err
}

// GetUnreadCount returns the number of unread messages for a user.
func (r *MessageRepository) GetUnreadCount(usrSeq int) (int, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM ALUMNI_MESSAGE
		WHERE AM_RECVR_SEQ = ? AND AM_READ_YN = 'N' AND AM_DEL_RECVR = 'N'
	`, usrSeq)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetConversations returns a list of conversation summaries for a user.
func (r *MessageRepository) GetConversations(usrSeq int) ([]model.ConversationSummary, error) {
	var conversations []model.ConversationSummary
	err := r.DB.Select(&conversations, `
		SELECT sub.other_seq, w.USR_NAME as other_name, m.AM_CONTENT as last_message,
			DATE_FORMAT(m.REG_DATE, '%Y-%m-%d %H:%i:%s') as last_date,
			(SELECT COUNT(*) FROM ALUMNI_MESSAGE
			 WHERE AM_RECVR_SEQ = ? AND AM_SENDER_SEQ = sub.other_seq
			 AND AM_READ_YN = 'N' AND AM_DEL_RECVR = 'N') as unread_count
		FROM (
			SELECT MAX(AM_SEQ) as last_seq,
				CASE WHEN AM_SENDER_SEQ = ? THEN AM_RECVR_SEQ ELSE AM_SENDER_SEQ END as other_seq
			FROM ALUMNI_MESSAGE
			WHERE (AM_SENDER_SEQ = ? OR AM_RECVR_SEQ = ?)
				AND ((AM_SENDER_SEQ = ? AND AM_DEL_SENDER = 'N') OR (AM_RECVR_SEQ = ? AND AM_DEL_RECVR = 'N'))
			GROUP BY CASE WHEN AM_SENDER_SEQ = ? THEN AM_RECVR_SEQ ELSE AM_SENDER_SEQ END
		) sub
		JOIN ALUMNI_MESSAGE m ON m.AM_SEQ = sub.last_seq
		JOIN WEO_MEMBER w ON w.USR_SEQ = sub.other_seq
		ORDER BY m.REG_DATE DESC
	`, usrSeq, usrSeq, usrSeq, usrSeq, usrSeq, usrSeq, usrSeq)
	if err != nil {
		return nil, err
	}
	return conversations, nil
}

// GetConversationMessages returns paginated messages between two users in chronological order.
func (r *MessageRepository) GetConversationMessages(usrSeq, otherSeq, page, size int) ([]model.Message, int, error) {
	var total int
	if err := r.DB.Get(&total, `
		SELECT COUNT(*) FROM ALUMNI_MESSAGE
		WHERE ((AM_SENDER_SEQ = ? AND AM_RECVR_SEQ = ? AND AM_DEL_SENDER = 'N')
			OR (AM_SENDER_SEQ = ? AND AM_RECVR_SEQ = ? AND AM_DEL_RECVR = 'N'))
	`, usrSeq, otherSeq, otherSeq, usrSeq); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	var messages []model.Message
	if err := r.DB.Select(&messages, `
		SELECT m.AM_SEQ, m.AM_SENDER_SEQ, m.AM_RECVR_SEQ, m.AM_CONTENT, m.AM_READ_YN,
			IFNULL(DATE_FORMAT(m.REG_DATE, '%Y-%m-%d %H:%i:%s'), '') AS REG_DATE,
			IFNULL(DATE_FORMAT(m.READ_DATE, '%Y-%m-%d %H:%i:%s'), '') AS READ_DATE,
			IFNULL(s.USR_NAME, '') AS SENDER_NAME,
			IFNULL(r2.USR_NAME, '') AS RECVR_NAME
		FROM ALUMNI_MESSAGE m
		LEFT JOIN WEO_MEMBER s ON s.USR_SEQ = m.AM_SENDER_SEQ
		LEFT JOIN WEO_MEMBER r2 ON r2.USR_SEQ = m.AM_RECVR_SEQ
		WHERE ((m.AM_SENDER_SEQ = ? AND m.AM_RECVR_SEQ = ? AND m.AM_DEL_SENDER = 'N')
			OR (m.AM_SENDER_SEQ = ? AND m.AM_RECVR_SEQ = ? AND m.AM_DEL_RECVR = 'N'))
		ORDER BY m.AM_SEQ ASC
		LIMIT ? OFFSET ?
	`, usrSeq, otherSeq, otherSeq, usrSeq, size, offset); err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// MarkConversationRead marks all unread messages from senderSeq to usrSeq as read.
func (r *MessageRepository) MarkConversationRead(usrSeq, senderSeq int) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_MESSAGE SET AM_READ_YN = 'Y', READ_DATE = NOW()
		WHERE AM_RECVR_SEQ = ? AND AM_SENDER_SEQ = ? AND AM_READ_YN = 'N' AND AM_DEL_RECVR = 'N'
	`, usrSeq, senderSeq)
	return err
}
