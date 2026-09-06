// message_report_repo.go — Authorized message evidence capture and moderation transactions.
package repository

import (
	"database/sql"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type MessageReportRepository struct{ DB *sqlx.DB }

func (r *MessageReportRepository) Create(reporter int, request model.MessageReportRequest) (int64, error) {
	// Select evidence on the server. A caller cannot report another user's inbox
	// or replace the original message with fabricated content.
	result, err := r.DB.Exec(`
		INSERT INTO ALUMNI_MESSAGE_REPORT
		(MESSAGE_ID, REPORTER_SEQ, REPORTED_SEQ, REASON, DETAILS, CONTENT_SNAPSHOT, CREATED_AT)
		SELECT AM_SEQ, ?, AM_SENDER_SEQ, ?, ?, AM_CONTENT, UTC_TIMESTAMP()
		FROM ALUMNI_MESSAGE WHERE AM_SEQ = ? AND AM_RECVR_SEQ = ?
		AND AM_SENDER_SEQ <> ? AND AM_VISIBLE_RECVR = 'Y' AND AM_DEL_RECVR = 'N'
		ON DUPLICATE KEY UPDATE REPORT_ID = LAST_INSERT_ID(REPORT_ID)
	`, reporter, request.Reason, request.Details, request.MessageID, reporter, reporter)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, sql.ErrNoRows
	}
	return id, nil
}

func (r *MessageReportRepository) List(status string, before int64) ([]model.MessageReport, error) {
	items := []model.MessageReport{}
	err := r.DB.Select(&items, `SELECT REPORT_ID, MESSAGE_ID, REPORTER_SEQ, REPORTED_SEQ,
		REASON, DETAILS, CONTENT_SNAPSHOT, STATUS, IFNULL(MODERATOR_NOTE, '') AS MODERATOR_NOTE,
		DATE_FORMAT(CREATED_AT, '%Y-%m-%dT%H:%i:%sZ') AS CREATED_AT
		FROM ALUMNI_MESSAGE_REPORT WHERE STATUS = ? AND (? = 0 OR REPORT_ID < ?)
		ORDER BY REPORT_ID DESC LIMIT 50`, status, before, before)
	return items, err
}

func (r *MessageReportRepository) Resolve(id int64, moderator int, resolution model.MessageReportResolution) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var messageID int64
	if err = tx.Get(&messageID, `SELECT MESSAGE_ID FROM ALUMNI_MESSAGE_REPORT
		WHERE REPORT_ID = ? AND STATUS = 'open' FOR UPDATE`, id); err != nil {
		return err
	}
	if resolution.Status == "removed" {
		// Keep the restricted report snapshot; remove content from both participants.
		if _, err = tx.Exec(`UPDATE ALUMNI_MESSAGE SET AM_CONTENT = ? WHERE AM_SEQ = ?`,
			"[운영정책 위반으로 삭제된 메시지입니다.]", messageID); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_MESSAGE_REPORT SET STATUS = ?, MODERATOR_SEQ = ?,
		MODERATOR_NOTE = ?, RESOLVED_AT = UTC_TIMESTAMP() WHERE REPORT_ID = ?`,
		resolution.Status, moderator, resolution.Note, id); err != nil {
		return err
	}
	return tx.Commit()
}
