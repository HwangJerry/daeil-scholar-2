package repository

import (
	"database/sql"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type FeedRepository struct {
	DB *sqlx.DB
}

func NewFeedRepository(db *sqlx.DB) *FeedRepository {
	return &FeedRepository{DB: db}
}

func (r *FeedRepository) GetNotices(cursor int, size int, heroSeq int, userSeq int) ([]model.NoticeItem, error) {
	args := []interface{}{}
	query := strings.Builder{}
	query.WriteString(`
		SELECT b.SEQ, b.SUBJECT, IFNULL(b.SUMMARY,'') AS SUMMARY, IFNULL(b.THUMBNAIL_URL,'') AS THUMBNAIL_URL,
		       b.REG_DATE, b.REG_NAME, b.HIT, b.IS_PINNED,
		       (SELECT COUNT(*) FROM WEO_BOARDLIKE
		        WHERE BBS_SEQ = b.SEQ AND OPEN_YN = 'Y') AS like_cnt,
		       (SELECT COUNT(*) FROM WEO_BOARDCOMAND
		        WHERE JOIN_SEQ = b.SEQ AND BC_TYPE = 'B' AND OPEN_YN = 'Y') AS comment_cnt,
		       IFNULL((SELECT 1 FROM WEO_BOARDLIKE
		               WHERE BBS_SEQ = b.SEQ AND USR_SEQ = ? AND OPEN_YN = 'Y'
		               LIMIT 1), 0) AS user_liked
		FROM WEO_BOARDBBS b
		WHERE b.GATE = 'NOTICE' AND b.OPEN_YN = 'Y'
	`)
	args = append(args, userSeq)
	if heroSeq > 0 {
		query.WriteString(" AND b.SEQ <> ?")
		args = append(args, heroSeq)
	}
	if cursor > 0 {
		query.WriteString(" AND b.SEQ < ?")
		args = append(args, cursor)
	}
	query.WriteString(" ORDER BY (b.IS_PINNED = 'Y') DESC, b.SEQ DESC LIMIT ?")
	args = append(args, size+1)

	var notices []model.NoticeItem
	if err := r.DB.Select(&notices, query.String(), args...); err != nil {
		return nil, err
	}
	return notices, nil
}

func (r *FeedRepository) GetHeroNotice() (*model.NoticeItem, error) {
	var notice model.NoticeItem
	err := r.DB.Get(&notice, `
		SELECT SEQ, SUBJECT, IFNULL(SUMMARY,'') AS SUMMARY, IFNULL(THUMBNAIL_URL,'') AS THUMBNAIL_URL,
		       REG_DATE, REG_NAME, HIT, IS_PINNED
		FROM WEO_BOARDBBS
		WHERE GATE = 'NOTICE' AND OPEN_YN = 'Y'
		ORDER BY (IS_PINNED = 'Y') DESC, SEQ DESC
		LIMIT 1
	`)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &notice, nil
}

func (r *FeedRepository) GetNoticeDetail(seq int) (*model.NoticeDetail, error) {
	var detail model.NoticeDetail
	err := r.DB.Get(&detail, `
		SELECT SEQ, SUBJECT, IFNULL(CONTENTS,'') AS CONTENTS,
		       IFNULL(CONTENTS_MD,'') AS CONTENTS_MD,
		       IFNULL(CONTENT_FORMAT,'LEGACY') AS CONTENT_FORMAT,
		       IFNULL(SUMMARY,'') AS SUMMARY, IFNULL(THUMBNAIL_URL,'') AS THUMBNAIL_URL,
		       REG_DATE, REG_NAME, HIT
		FROM WEO_BOARDBBS
		WHERE SEQ = ? AND GATE = 'NOTICE' AND OPEN_YN = 'Y'
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

func (r *FeedRepository) IncrementHit(seq int) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_BOARDBBS SET HIT = HIT + 1 WHERE SEQ = ?
	`, seq)
	return err
}

func (r *FeedRepository) GetLikeCount(seq int) (int, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM WEO_BOARDLIKE WHERE BBS_SEQ = ? AND OPEN_YN = 'Y'
	`, seq)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *FeedRepository) GetCommentCount(seq int) (int, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM WEO_BOARDCOMAND WHERE JOIN_SEQ = ? AND BC_TYPE = 'B' AND OPEN_YN = 'Y'
	`, seq)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *FeedRepository) GetPrevPost(seq int) (*model.PostSibling, error) {
	var sibling model.PostSibling
	err := r.DB.Get(&sibling, `
		SELECT SEQ, SUBJECT FROM WEO_BOARDBBS
		WHERE SEQ < ? AND GATE = 'NOTICE' AND OPEN_YN = 'Y'
		ORDER BY SEQ DESC LIMIT 1
	`, seq)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &sibling, nil
}

func (r *FeedRepository) GetNextPost(seq int) (*model.PostSibling, error) {
	var sibling model.PostSibling
	err := r.DB.Get(&sibling, `
		SELECT SEQ, SUBJECT FROM WEO_BOARDBBS
		WHERE SEQ > ? AND GATE = 'NOTICE' AND OPEN_YN = 'Y'
		ORDER BY SEQ ASC LIMIT 1
	`, seq)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &sibling, nil
}

func (r *FeedRepository) GetFilesByPost(seq int) ([]model.FileRecord, error) {
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
