// message_repo.go — Database access layer for alumni direct messaging
package repository

import (
	"database/sql"
	"time"

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

func (r *MessageRepository) FindAcceptedMessage(senderSeq int, clientMessageID string) (*model.SendMessageResponse, error) {
	var accepted model.SendMessageResponse
	if err := r.DB.Get(&accepted, `
		SELECT AM_SEQ, AM_CLIENT_MESSAGE_ID, AM_VISIBLE_RECVR,
			DATE_FORMAT(REG_DATE, '%Y-%m-%dT%H:%i:%sZ') AS CREATED_AT
		FROM ALUMNI_MESSAGE
		WHERE AM_SENDER_SEQ = ? AND AM_CLIENT_MESSAGE_ID = ?
		LIMIT 1
	`, senderSeq, clientMessageID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	accepted.Status = "accepted"
	return &accepted, nil
}

// AcceptMessage inserts a message once per sender/client ID and returns the
// original accepted row for every replay.
func (r *MessageRepository) AcceptMessage(senderSeq, recvrSeq int, clientMessageID, content string) (*model.SendMessageResponse, error) {
	result, err := r.DB.Exec(`
		INSERT INTO ALUMNI_MESSAGE (
			AM_SENDER_SEQ, AM_RECVR_SEQ, AM_CLIENT_MESSAGE_ID,
			AM_CONTENT, AM_READ_YN, AM_VISIBLE_RECVR,
			AM_SUPPRESSION_REASON, PURGE_AT, REG_DATE
		)
		SELECT ?, ?, ?, ?, 'N',
			CASE WHEN block_state.blocked = 1 THEN 'N' ELSE 'Y' END,
			CASE WHEN block_state.blocked = 1 THEN 'recipient_blocked' ELSE NULL END,
			CASE WHEN block_state.blocked = 1 THEN DATE_ADD(UTC_TIMESTAMP(), INTERVAL 1 MONTH) ELSE NULL END,
			UTC_TIMESTAMP()
		FROM (
			SELECT EXISTS(
				SELECT 1 FROM ALUMNI_MEMBER_BLOCK
				WHERE BLOCKER_USR_SEQ = ? AND BLOCKED_USR_SEQ = ?
			) AS blocked
		) AS block_state
		ON DUPLICATE KEY UPDATE AM_SEQ = LAST_INSERT_ID(AM_SEQ)
	`, senderSeq, recvrSeq, clientMessageID, content, recvrSeq, senderSeq)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	var accepted model.SendMessageResponse
	if err := r.DB.Get(&accepted, `
		SELECT AM_SEQ, AM_CLIENT_MESSAGE_ID, AM_VISIBLE_RECVR,
			DATE_FORMAT(REG_DATE, '%Y-%m-%dT%H:%i:%sZ') AS CREATED_AT
		FROM ALUMNI_MESSAGE
		WHERE AM_SENDER_SEQ = ? AND AM_CLIENT_MESSAGE_ID = ?
		LIMIT 1
	`, senderSeq, clientMessageID); err != nil {
		return nil, err
	}
	accepted.Status = "accepted"
	accepted.WasCreated = affected == 1
	return &accepted, nil
}

func (r *MessageRepository) IsApprovedAlumni(usrSeq int) (bool, error) {
	var exists bool
	if err := r.DB.Get(&exists, `
		SELECT EXISTS(
			SELECT 1 FROM ALUMNI_VERIFICATION
			WHERE USR_SEQ = ? AND STATUS = 'approved'
		)
	`, usrSeq); err != nil {
		return false, err
	}
	return exists, nil
}

// GetInbox returns received messages for a user, paginated.
func (r *MessageRepository) GetInbox(usrSeq int, page int, size int) ([]model.Message, int, error) {
	var total int
	if err := r.DB.Get(&total, `
		SELECT COUNT(*) FROM ALUMNI_MESSAGE
		WHERE AM_RECVR_SEQ = ? AND AM_DEL_RECVR = 'N' AND AM_VISIBLE_RECVR = 'Y'
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
		WHERE am.AM_RECVR_SEQ = ? AND am.AM_DEL_RECVR = 'N' AND am.AM_VISIBLE_RECVR = 'Y'
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

// MarkAsRead marks a message as read and reports the sender if the row changed.
func (r *MessageRepository) MarkAsRead(amSeq int, usrSeq int) (int, bool, error) {
	result, err := r.DB.Exec(`
		UPDATE ALUMNI_MESSAGE SET AM_READ_YN = 'Y', READ_DATE = NOW()
		WHERE AM_SEQ = ? AND AM_RECVR_SEQ = ? AND AM_READ_YN = 'N' AND AM_VISIBLE_RECVR = 'Y'
	`, amSeq, usrSeq)
	if err != nil {
		return 0, false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, false, err
	}
	if affected == 0 {
		return 0, false, nil
	}

	var senderSeq int
	if err := r.DB.Get(&senderSeq, `
		SELECT AM_SENDER_SEQ FROM ALUMNI_MESSAGE
		WHERE AM_SEQ = ? AND AM_RECVR_SEQ = ? AND AM_VISIBLE_RECVR = 'Y'
	`, amSeq, usrSeq); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	return senderSeq, true, nil
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
		WHERE AM_RECVR_SEQ = ? AND AM_READ_YN = 'N' AND AM_DEL_RECVR = 'N' AND AM_VISIBLE_RECVR = 'Y'
	`, usrSeq)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetConversations returns a list of conversation summaries for a user.
func (r *MessageRepository) GetConversations(usrSeq int, beforeCreatedAt *time.Time, beforeMessageID int64, limit int) ([]model.ConversationSummary, error) {
	query := `
		SELECT sub.other_seq AS USER_SEQ,
			COALESCE(NULLIF(CONVERT(w.USR_NAME USING utf8mb4), _utf8mb4''), _utf8mb4'탈퇴한 회원') AS NAME,
			m.AM_CONTENT AS LAST_MESSAGE,
			DATE_FORMAT(m.REG_DATE, '%Y-%m-%dT%H:%i:%sZ') AS LAST_MESSAGE_AT,
			(SELECT COUNT(*) FROM ALUMNI_MESSAGE
			 WHERE AM_RECVR_SEQ = ? AND AM_SENDER_SEQ = sub.other_seq
			 AND AM_READ_YN = 'N' AND AM_DEL_RECVR = 'N' AND AM_VISIBLE_RECVR = 'Y') AS UNREAD_COUNT,
			EXISTS(
				SELECT 1 FROM ALUMNI_MEMBER_BLOCK b
				WHERE b.BLOCKER_USR_SEQ = ? AND b.BLOCKED_USR_SEQ = sub.other_seq
			) AS BLOCKED_BY_ME,
			m.REG_DATE AS CURSOR_CREATED_AT,
			m.AM_SEQ AS CURSOR_MESSAGE_ID
		FROM (
			SELECT DISTINCT
				CASE WHEN AM_SENDER_SEQ = ? THEN AM_RECVR_SEQ ELSE AM_SENDER_SEQ END AS other_seq
			FROM ALUMNI_MESSAGE
			WHERE (AM_SENDER_SEQ = ? AND AM_DEL_SENDER = 'N')
				OR (AM_RECVR_SEQ = ? AND AM_DEL_RECVR = 'N' AND AM_VISIBLE_RECVR = 'Y')
		) sub
		JOIN ALUMNI_MESSAGE m ON m.AM_SEQ = (
			SELECT latest.AM_SEQ
			FROM ALUMNI_MESSAGE latest
			WHERE (latest.AM_SENDER_SEQ = ? AND latest.AM_RECVR_SEQ = sub.other_seq AND latest.AM_DEL_SENDER = 'N')
				OR (latest.AM_SENDER_SEQ = sub.other_seq AND latest.AM_RECVR_SEQ = ?
					AND latest.AM_DEL_RECVR = 'N' AND latest.AM_VISIBLE_RECVR = 'Y')
			ORDER BY latest.REG_DATE DESC, latest.AM_SEQ DESC
			LIMIT 1
		)
		LEFT JOIN WEO_MEMBER w ON w.USR_SEQ = sub.other_seq
	`
	args := []interface{}{usrSeq, usrSeq, usrSeq, usrSeq, usrSeq, usrSeq, usrSeq}
	if beforeCreatedAt != nil {
		beforeUTC := beforeCreatedAt.UTC()
		query += `
		WHERE (m.REG_DATE < ? OR (m.REG_DATE = ? AND m.AM_SEQ < ?))`
		args = append(args, beforeUTC, beforeUTC, beforeMessageID)
	}
	query += `
		ORDER BY m.REG_DATE DESC, m.AM_SEQ DESC
		LIMIT ?`
	args = append(args, limit)

	var conversations []model.ConversationSummary
	err := r.DB.Select(&conversations, query, args...)
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
			OR (AM_SENDER_SEQ = ? AND AM_RECVR_SEQ = ? AND AM_DEL_RECVR = 'N' AND AM_VISIBLE_RECVR = 'Y'))
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
			OR (m.AM_SENDER_SEQ = ? AND m.AM_RECVR_SEQ = ? AND m.AM_DEL_RECVR = 'N' AND m.AM_VISIBLE_RECVR = 'Y'))
		ORDER BY m.AM_SEQ ASC
		LIMIT ? OFFSET ?
	`, usrSeq, otherSeq, otherSeq, usrSeq, size, offset); err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

func (r *MessageRepository) GetCanonicalConversationMessages(usrSeq, otherSeq int, beforeCreatedAt *time.Time, beforeMessageID int64, limit int) ([]model.Message, error) {
	query := `
		SELECT m.AM_SEQ,
			COALESCE(m.AM_CLIENT_MESSAGE_ID, CONCAT('legacy-', m.AM_SEQ)) AS AM_CLIENT_MESSAGE_ID,
			m.AM_SENDER_SEQ, m.AM_RECVR_SEQ,
			m.AM_CONTENT, m.AM_READ_YN,
			DATE_FORMAT(m.REG_DATE, '%Y-%m-%dT%H:%i:%sZ') AS REG_DATE,
			IF(m.READ_DATE IS NULL, '', DATE_FORMAT(m.READ_DATE, '%Y-%m-%dT%H:%i:%sZ')) AS READ_DATE,
			COALESCE(NULLIF(s.USR_NAME, ''), '탈퇴한 회원') AS SENDER_NAME,
			COALESCE(NULLIF(r2.USR_NAME, ''), '탈퇴한 회원') AS RECVR_NAME
		FROM ALUMNI_MESSAGE m
		LEFT JOIN WEO_MEMBER s ON s.USR_SEQ = m.AM_SENDER_SEQ
		LEFT JOIN WEO_MEMBER r2 ON r2.USR_SEQ = m.AM_RECVR_SEQ
		WHERE ((m.AM_SENDER_SEQ = ? AND m.AM_RECVR_SEQ = ? AND m.AM_DEL_SENDER = 'N')
			OR (m.AM_SENDER_SEQ = ? AND m.AM_RECVR_SEQ = ? AND m.AM_DEL_RECVR = 'N' AND m.AM_VISIBLE_RECVR = 'Y'))`
	args := []interface{}{usrSeq, otherSeq, otherSeq, usrSeq}
	if beforeCreatedAt != nil {
		beforeUTC := beforeCreatedAt.UTC()
		query += `
		AND (m.REG_DATE < ? OR (m.REG_DATE = ? AND m.AM_SEQ < ?))`
		args = append(args, beforeUTC, beforeUTC, beforeMessageID)
	}
	query += `
		ORDER BY m.REG_DATE DESC, m.AM_SEQ DESC
		LIMIT ?`
	args = append(args, limit)

	var messages []model.Message
	if err := r.DB.Select(&messages, query, args...); err != nil {
		return nil, err
	}
	return messages, nil
}

// MarkConversationRead marks all unread messages from senderSeq to usrSeq as read.
func (r *MessageRepository) MarkConversationRead(usrSeq, senderSeq int, throughMessageID int64) (bool, error) {
	result, err := r.DB.Exec(`
		UPDATE ALUMNI_MESSAGE AS m
		JOIN ALUMNI_MESSAGE AS boundary ON boundary.AM_SEQ = ?
			AND ((boundary.AM_SENDER_SEQ = ? AND boundary.AM_RECVR_SEQ = ?
				AND boundary.AM_DEL_RECVR = 'N' AND boundary.AM_VISIBLE_RECVR = 'Y')
				OR (boundary.AM_SENDER_SEQ = ? AND boundary.AM_RECVR_SEQ = ?
					AND boundary.AM_DEL_SENDER = 'N'))
		SET m.AM_READ_YN = 'Y', m.READ_DATE = NOW()
		WHERE m.AM_RECVR_SEQ = ? AND m.AM_SENDER_SEQ = ?
			AND m.AM_SEQ <= boundary.AM_SEQ AND m.AM_READ_YN = 'N'
			AND m.AM_DEL_RECVR = 'N' AND m.AM_VISIBLE_RECVR = 'Y'
	`, throughMessageID, senderSeq, usrSeq, usrSeq, senderSeq, usrSeq, senderSeq)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}
