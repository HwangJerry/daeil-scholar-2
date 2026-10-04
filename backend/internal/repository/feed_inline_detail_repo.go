// feed_inline_detail_repo.go — Batched reads that let a feed page carry each post's expanded detail
package repository

import (
	"database/sql"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// GetNoticeBodies reads the raw bodies of the given posts in one query. The
// columns and defaults match GetNoticeDetail so the presenter decodes them the
// same way.
func (r *FeedRepository) GetNoticeBodies(seqs []int) ([]model.NoticeBody, error) {
	bodies := make([]model.NoticeBody, 0, len(seqs))
	if len(seqs) == 0 {
		return bodies, nil
	}
	if err := r.selectIn(&bodies, `
		SELECT SEQ, IFNULL(CONTENTS,'') AS CONTENTS,
		       IFNULL(CONTENT_FORMAT,'LEGACY') AS CONTENT_FORMAT
		FROM WEO_BOARDBBS
		WHERE SEQ IN (?)
	`, seqs); err != nil {
		return nil, err
	}
	return bodies, nil
}

// GetCommentsByPosts reads the visible comments of the given posts in one
// query, with the same columns, filter and per-post order (newest first) as
// CommentRepository.GetComments.
func (r *FeedRepository) GetCommentsByPosts(seqs []int) ([]model.Comment, error) {
	comments := make([]model.Comment, 0)
	if len(seqs) == 0 {
		return comments, nil
	}
	if err := r.selectIn(&comments, `
		SELECT SEQ AS BC_SEQ, JOIN_SEQ, USR_SEQ, IFNULL(NICKNAME,'') AS NICKNAME,
		       IFNULL(CONTENTS,'') AS CONTENTS, DATE_FORMAT(REG_DATE, '%Y-%m-%d %H:%i') AS REG_DATE
		FROM WEO_BOARDCOMAND
		WHERE JOIN_SEQ IN (?) AND BC_TYPE = 'B' AND OPEN_YN = 'Y'
		ORDER BY JOIN_SEQ, SEQ DESC
	`, seqs); err != nil {
		return nil, err
	}
	return comments, nil
}

// GetFilesByPosts reads the open attachments of the given posts in one query,
// with the same columns and filter as GetFilesByPost.
func (r *FeedRepository) GetFilesByPosts(seqs []int) ([]model.FileRecord, error) {
	files := make([]model.FileRecord, 0)
	if len(seqs) == 0 {
		return files, nil
	}
	if err := r.selectIn(&files, `
		SELECT F_SEQ, IFNULL(F_GATE,'') AS F_GATE, F_JOIN_SEQ,
		       IFNULL(TYPE_NAME,'') AS TYPE_NAME, IFNULL(FILE_NAME,'') AS FILE_NAME,
		       IFNULL(FILE_SIZE,'0') AS FILE_SIZE, IFNULL(FILE_PATH,'') AS FILE_PATH,
		       IFNULL(FILE_ORG_NAME,'') AS FILE_ORG_NAME, IFNULL(OPEN_YN,'N') AS OPEN_YN
		FROM WEO_FILES
		WHERE F_JOIN_SEQ IN (?) AND F_GATE = 'BB' AND OPEN_YN = 'Y'
		ORDER BY F_JOIN_SEQ, F_SEQ
	`, seqs); err != nil {
		return nil, err
	}
	return files, nil
}

// GetLikeStats reads like counts and the user's own like for the given posts in
// one query. Posts without likes are absent from the result.
func (r *FeedRepository) GetLikeStats(seqs []int, userSeq int) ([]model.NoticeLikeStats, error) {
	stats := make([]model.NoticeLikeStats, 0, len(seqs))
	if len(seqs) == 0 {
		return stats, nil
	}
	query, args, err := sqlx.In(`
		SELECT BBS_SEQ, COUNT(*) AS like_cnt, MAX(USR_SEQ = ?) AS user_liked
		FROM WEO_BOARDLIKE
		WHERE BBS_SEQ IN (?) AND OPEN_YN = 'Y'
		GROUP BY BBS_SEQ
	`, userSeq, seqs)
	if err != nil {
		return nil, err
	}
	if err := r.DB.Select(&stats, r.DB.Rebind(query), args...); err != nil {
		return nil, err
	}
	return stats, nil
}

// GetPublishedNoticeHit returns the view count of a published notice, with the
// same visibility filter as GetNoticeDetail. found is false when the post is
// missing or unpublished.
func (r *FeedRepository) GetPublishedNoticeHit(seq int) (hit int, found bool, err error) {
	err = r.DB.Get(&hit, `
		SELECT HIT FROM WEO_BOARDBBS
		WHERE SEQ = ? AND GATE = 'NOTICE' AND OPEN_YN = 'Y'
		LIMIT 1
	`, seq)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return hit, true, nil
}

func (r *FeedRepository) selectIn(dest interface{}, query string, seqs []int) error {
	expanded, args, err := sqlx.In(query, seqs)
	if err != nil {
		return err
	}
	return r.DB.Select(dest, r.DB.Rebind(expanded), args...)
}
