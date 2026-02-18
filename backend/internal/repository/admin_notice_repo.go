// Admin notice repository — CRUD queries for WEO_BOARDBBS admin operations
package repository

import (
	"database/sql"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AdminNoticeRepository struct {
	DB *sqlx.DB
}

func NewAdminNoticeRepository(db *sqlx.DB) *AdminNoticeRepository {
	return &AdminNoticeRepository{DB: db}
}

func (r *AdminNoticeRepository) GetNotices(page, size int, keyword string) ([]model.AdminNoticeRow, int, error) {
	args := []interface{}{}
	where := "WHERE GATE = 'NOTICE'"
	if keyword != "" {
		where += " AND SUBJECT LIKE ?"
		args = append(args, keyword+"%")
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.DB.Get(&total, "SELECT COUNT(*) FROM WEO_BOARDBBS "+where, countArgs...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	query := `SELECT SEQ, SUBJECT, REG_DATE, REG_NAME, HIT, OPEN_YN, IS_PINNED, CONTENT_FORMAT
		FROM WEO_BOARDBBS ` + where + ` ORDER BY SEQ DESC LIMIT ? OFFSET ?`
	args = append(args, size, offset)

	var rows []model.AdminNoticeRow
	if err := r.DB.Select(&rows, query, args...); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *AdminNoticeRepository) GetNoticeForEdit(seq int) (*model.NoticeDetail, error) {
	var detail model.NoticeDetail
	err := r.DB.Get(&detail, `
		SELECT SEQ, SUBJECT, CONTENTS, CONTENTS_MD, CONTENT_FORMAT, SUMMARY, THUMBNAIL_URL, REG_DATE, REG_NAME, HIT
		FROM WEO_BOARDBBS WHERE SEQ = ? AND GATE = 'NOTICE' LIMIT 1
	`, seq)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &detail, nil
}

func (r *AdminNoticeRepository) InsertNotice(n *model.AdminNoticeInsert) (int, error) {
	res, err := r.DB.Exec(`
		INSERT INTO WEO_BOARDBBS
			(GATE, SUBJECT, CONTENTS, CONTENTS_MD, CONTENT_FORMAT, SUMMARY, THUMBNAIL_URL, IS_PINNED, OPEN_YN, REG_NAME, REG_DATE, HIT)
		VALUES ('NOTICE', ?, ?, ?, 'MARKDOWN', ?, ?, ?, 'Y', ?, NOW(), 0)
	`, n.Subject, n.Contents, n.ContentsMD, n.Summary, n.ThumbnailURL, n.IsPinned, n.RegName)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *AdminNoticeRepository) UpdateNotice(seq int, n *model.AdminNoticeInsert) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_BOARDBBS
		SET SUBJECT = ?, CONTENTS = ?, CONTENTS_MD = ?, CONTENT_FORMAT = 'MARKDOWN',
		    SUMMARY = ?, THUMBNAIL_URL = ?, IS_PINNED = ?
		WHERE SEQ = ? AND GATE = 'NOTICE'
	`, n.Subject, n.Contents, n.ContentsMD, n.Summary, n.ThumbnailURL, n.IsPinned, seq)
	return err
}

func (r *AdminNoticeRepository) DeleteNotice(seq int) error {
	_, err := r.DB.Exec(`UPDATE WEO_BOARDBBS SET OPEN_YN = 'N' WHERE SEQ = ? AND GATE = 'NOTICE'`, seq)
	return err
}

func (r *AdminNoticeRepository) TogglePin(seq int) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_BOARDBBS
		SET IS_PINNED = CASE WHEN IS_PINNED = 'Y' THEN 'N' ELSE 'Y' END
		WHERE SEQ = ? AND GATE = 'NOTICE'
	`, seq)
	return err
}

func (r *AdminNoticeRepository) CountNotices() (int, error) {
	var c int
	err := r.DB.Get(&c, `SELECT COUNT(*) FROM WEO_BOARDBBS WHERE GATE = 'NOTICE'`)
	return c, err
}
