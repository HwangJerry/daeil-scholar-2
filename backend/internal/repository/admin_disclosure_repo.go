// Admin disclosure repository — CRUD queries for WEO_BOARDBBS GATE='DISCLOSURE'
package repository

import (
	"database/sql"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AdminDisclosureRepository struct {
	DB *sqlx.DB
}

func NewAdminDisclosureRepository(db *sqlx.DB) *AdminDisclosureRepository {
	return &AdminDisclosureRepository{DB: db}
}

func (r *AdminDisclosureRepository) GetDisclosures(page, size int, keyword string) ([]model.AdminDisclosureRow, int, error) {
	args := []interface{}{}
	where := "WHERE GATE = 'DISCLOSURE'"
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
	query := `SELECT SEQ, SUBJECT, REG_DATE, REG_NAME, HIT, OPEN_YN, CONTENT_FORMAT
		FROM WEO_BOARDBBS ` + where + ` ORDER BY SEQ DESC LIMIT ? OFFSET ?`
	args = append(args, size, offset)

	var rows []model.AdminDisclosureRow
	if err := r.DB.Select(&rows, query, args...); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *AdminDisclosureRepository) GetDisclosureForEdit(seq int) (*model.NoticeDetail, error) {
	var detail model.NoticeDetail
	err := r.DB.Get(&detail, `
		SELECT SEQ, SUBJECT, CONTENTS, CONTENTS_MD, CONTENT_FORMAT, SUMMARY, THUMBNAIL_URL, REG_DATE, REG_NAME, HIT, IS_PINNED
		FROM WEO_BOARDBBS WHERE SEQ = ? AND GATE = 'DISCLOSURE' LIMIT 1
	`, seq)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &detail, nil
}

func (r *AdminDisclosureRepository) InsertDisclosure(d *model.AdminDisclosureInsert) (int, error) {
	// WEO_BOARDBBS.SEQ is not AUTO_INCREMENT (legacy table); generate next SEQ manually.
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
			 STEP, USR_SEQ, REG_NAME, REG_DATE, HIT, LIKE_CNT)
		VALUES (?, 'DISCLOSURE', 0, 0, 0,
		        ?, ?, ?, 'MARKDOWN', 'H',
		        ?, ?, 'N', 'Y', 'Y', 'N',
		        'U', ?, ?, NOW(), 0, 0)
	`, nextSeq, d.Subject, d.Contents, d.ContentsMD,
		d.Summary, d.ThumbnailURL,
		d.USRSeq, d.RegName)
	if err != nil {
		return 0, err
	}
	return nextSeq, nil
}

func (r *AdminDisclosureRepository) UpdateDisclosure(seq int, d *model.AdminDisclosureInsert) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_BOARDBBS
		SET SUBJECT = ?, CONTENTS = ?, CONTENTS_MD = ?, CONTENT_FORMAT = 'MARKDOWN',
		    SUMMARY = ?, THUMBNAIL_URL = ?
		WHERE SEQ = ? AND GATE = 'DISCLOSURE'
	`, d.Subject, d.Contents, d.ContentsMD, d.Summary, d.ThumbnailURL, seq)
	return err
}

func (r *AdminDisclosureRepository) DeleteDisclosure(seq int) error {
	_, err := r.DB.Exec(`UPDATE WEO_BOARDBBS SET OPEN_YN = 'N' WHERE SEQ = ? AND GATE = 'DISCLOSURE'`, seq)
	return err
}

func (r *AdminDisclosureRepository) CountDisclosures() (int, error) {
	var c int
	err := r.DB.Get(&c, `SELECT COUNT(*) FROM WEO_BOARDBBS WHERE GATE = 'DISCLOSURE'`)
	return c, err
}
