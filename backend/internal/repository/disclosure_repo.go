// Disclosure repository — public read-only queries for WEO_BOARDBBS GATE='DISCLOSURE'
package repository

import (
	"database/sql"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type DisclosureRepository struct {
	DB *sqlx.DB
}

func NewDisclosureRepository(db *sqlx.DB) *DisclosureRepository {
	return &DisclosureRepository{DB: db}
}

// ListDisclosures returns up to size+1 rows; the caller trims and uses the extra row to detect hasMore.
func (r *DisclosureRepository) ListDisclosures(cursor, size int) ([]model.DisclosureItem, error) {
	args := []interface{}{}
	query := `
		SELECT b.SEQ, b.SUBJECT, IFNULL(b.SUMMARY,'') AS SUMMARY,
		       b.REG_DATE, b.REG_NAME, b.HIT,
		       (SELECT COUNT(*) FROM WEO_FILES
		        WHERE F_JOIN_SEQ = b.SEQ AND F_GATE = 'BB' AND OPEN_YN = 'Y') AS attachment_count
		FROM WEO_BOARDBBS b
		WHERE b.GATE = 'DISCLOSURE' AND b.OPEN_YN = 'Y'
	`
	if cursor > 0 {
		query += " AND b.SEQ < ?"
		args = append(args, cursor)
	}
	query += " ORDER BY b.SEQ DESC LIMIT ?"
	args = append(args, size+1)

	var items []model.DisclosureItem
	if err := r.DB.Select(&items, query, args...); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *DisclosureRepository) GetDisclosureDetail(seq int) (*model.NoticeDetail, error) {
	var detail model.NoticeDetail
	err := r.DB.Get(&detail, `
		SELECT SEQ, SUBJECT, IFNULL(CONTENTS,'') AS CONTENTS,
		       IFNULL(CONTENTS_MD,'') AS CONTENTS_MD,
		       IFNULL(CONTENT_FORMAT,'LEGACY') AS CONTENT_FORMAT,
		       IFNULL(SUMMARY,'') AS SUMMARY, IFNULL(THUMBNAIL_URL,'') AS THUMBNAIL_URL,
		       REG_DATE, REG_NAME, HIT, IS_PINNED
		FROM WEO_BOARDBBS
		WHERE SEQ = ? AND GATE = 'DISCLOSURE' AND OPEN_YN = 'Y'
		LIMIT 1
	`, seq)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &detail, nil
}

func (r *DisclosureRepository) IncrementHit(seq int) error {
	_, err := r.DB.Exec(`UPDATE WEO_BOARDBBS SET HIT = HIT + 1 WHERE SEQ = ? AND GATE = 'DISCLOSURE'`, seq)
	return err
}

func (r *DisclosureRepository) GetFilesByDisclosure(seq int) ([]model.FileRecord, error) {
	files := make([]model.FileRecord, 0)
	err := r.DB.Select(&files, `
		SELECT F_SEQ, IFNULL(F_GATE,'') AS F_GATE, F_JOIN_SEQ,
		       IFNULL(TYPE_NAME,'') AS TYPE_NAME, IFNULL(FILE_NAME,'') AS FILE_NAME,
		       IFNULL(FILE_SIZE,'0') AS FILE_SIZE, IFNULL(FILE_PATH,'') AS FILE_PATH,
		       IFNULL(FILE_ORG_NAME,'') AS FILE_ORG_NAME, IFNULL(OPEN_YN,'N') AS OPEN_YN
		FROM WEO_FILES
		WHERE F_JOIN_SEQ = ? AND F_GATE = 'BB' AND OPEN_YN = 'Y'
	`, seq)
	if err != nil {
		return nil, err
	}
	return files, nil
}
