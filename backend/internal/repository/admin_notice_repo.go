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

// GetNotices lists NOTICE posts newest first. categorySeq > 0 keeps only that
// category; filtering by the default also matches posts with no category.
func (r *AdminNoticeRepository) GetNotices(page, size int, keyword string, categorySeq int) ([]model.AdminNoticeRow, int, error) {
	args := []interface{}{}
	from := "FROM WEO_BOARDBBS b" + feedCategoryJoin
	where := " WHERE b.GATE = 'NOTICE'"
	if keyword != "" {
		where += " AND b.SUBJECT LIKE ?"
		args = append(args, keyword+"%")
	}
	if categorySeq > 0 {
		where += " AND " + feedCategorySeqExpr + " = ?"
		args = append(args, categorySeq)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.DB.Get(&total, "SELECT COUNT(*) "+from+where, countArgs...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	query := `SELECT b.SEQ, b.SUBJECT, b.REG_DATE, b.REG_NAME, b.HIT, b.OPEN_YN, b.IS_PINNED, b.CONTENT_FORMAT,
		` + feedCategorySeqExpr + ` AS category_seq, IFNULL(fc.FC_NAME, IFNULL(fd.FC_NAME, '공지')) AS category_name
		` + from + where + ` ORDER BY b.SEQ DESC LIMIT ? OFFSET ?`
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
		SELECT b.SEQ, b.SUBJECT, b.CONTENTS, b.CONTENTS_MD, b.CONTENT_FORMAT, b.SUMMARY, b.THUMBNAIL_URL,
		       b.REG_DATE, b.REG_NAME, b.HIT, b.IS_PINNED,`+feedCategoryColumns+`,
		       `+feedCategorySeqExpr+` AS category_seq
		FROM WEO_BOARDBBS b`+feedCategoryJoin+`
		WHERE b.SEQ = ? AND b.GATE = 'NOTICE' LIMIT 1
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
	// WEO_BOARDBBS.SEQ is not AUTO_INCREMENT (legacy table); generate next SEQ manually.
	// Use sql.NullInt64 to handle the case where the table is empty (MAX returns NULL).
	var maxSeq sql.NullInt64
	if err := r.DB.Get(&maxSeq, `SELECT MAX(SEQ) FROM WEO_BOARDBBS`); err != nil {
		return 0, err
	}
	nextSeq := int(maxSeq.Int64) + 1

	_, err := r.DB.Exec(`
		INSERT INTO WEO_BOARDBBS
			(SEQ, GATE, P_ID, B_NO, R_NO,
			 SUBJECT, CONTENTS, CONTENTS_MD, CONTENT_FORMAT, CONTENTS_TYPE,
			 SUMMARY, THUMBNAIL_URL, IS_PINNED, OPEN_YN, OPEN_TYPE, REPLY_MAIL,
			 STEP, USR_SEQ, REG_NAME, REG_DATE, HIT, LIKE_CNT, FEED_CATEGORY_SEQ)
		VALUES (?, 'NOTICE', 0, 0, 0,
		        ?, ?, ?, 'MARKDOWN', 'H',
		        ?, ?, ?, 'Y', 'Y', 'N',
		        'U', ?, ?, NOW(), 0, 0, ?)
	`, nextSeq, n.Subject, n.Contents, n.ContentsMD,
		n.Summary, n.ThumbnailURL, n.IsPinned,
		n.USRSeq, n.RegName, n.FeedCategorySeq)
	if err != nil {
		return 0, err
	}
	return nextSeq, nil
}

func (r *AdminNoticeRepository) UpdateNotice(seq int, n *model.AdminNoticeInsert) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_BOARDBBS
		SET SUBJECT = ?, CONTENTS = ?, CONTENTS_MD = ?, CONTENT_FORMAT = 'MARKDOWN',
		    SUMMARY = ?, THUMBNAIL_URL = ?, IS_PINNED = ?,
		    FEED_CATEGORY_SEQ = IFNULL(?, FEED_CATEGORY_SEQ)
		WHERE SEQ = ? AND GATE = 'NOTICE'
	`, n.Subject, n.Contents, n.ContentsMD, n.Summary, n.ThumbnailURL, n.IsPinned, n.FeedCategorySeq, seq)
	return err
}

// UpdateNoticeCategory changes only a post's category, so legacy (HTML) posts
// can be reclassified without touching their content.
func (r *AdminNoticeRepository) UpdateNoticeCategory(seq, categorySeq int) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_BOARDBBS SET FEED_CATEGORY_SEQ = ? WHERE SEQ = ? AND GATE = 'NOTICE'
	`, categorySeq, seq)
	return err
}

func (r *AdminNoticeRepository) DeleteNotice(seq int) error {
	_, err := r.DB.Exec(`UPDATE WEO_BOARDBBS SET OPEN_YN = 'N' WHERE SEQ = ? AND GATE = 'NOTICE'`, seq)
	return err
}

func (r *AdminNoticeRepository) TogglePin(seq int) error {
	// Step 1: unpin all other notices to enforce single-pin constraint
	if _, err := r.DB.Exec(`
		UPDATE WEO_BOARDBBS SET IS_PINNED = 'N'
		WHERE GATE = 'NOTICE' AND SEQ <> ?
	`, seq); err != nil {
		return err
	}
	// Step 2: toggle this notice
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
