// comment_repo.go — Repository for WEO_BOARDCOMAND table operations
package repository

import (
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// CommentRepository handles comment-related database operations.
type CommentRepository struct {
	DB *sqlx.DB
}

// NewCommentRepository creates a new CommentRepository.
func NewCommentRepository(db *sqlx.DB) *CommentRepository {
	return &CommentRepository{DB: db}
}

// GetComments returns all visible comments for a post, ordered by SEQ descending (newest first).
func (r *CommentRepository) GetComments(joinSeq int) ([]model.Comment, error) {
	comments := make([]model.Comment, 0)
	err := r.DB.Select(&comments, `
		SELECT SEQ AS BC_SEQ, JOIN_SEQ, USR_SEQ, IFNULL(NICKNAME,'') AS NICKNAME,
		       IFNULL(CONTENTS,'') AS CONTENTS, REG_DATE
		FROM WEO_BOARDCOMAND
		WHERE JOIN_SEQ = ? AND BC_TYPE = 'B' AND OPEN_YN = 'Y'
		ORDER BY SEQ DESC
	`, joinSeq)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

// InsertComment creates a new comment and returns the last insert ID.
func (r *CommentRepository) InsertComment(joinSeq int, usrSeq int, regName string, contents string) (int64, error) {
	result, err := r.DB.Exec(`
		INSERT INTO WEO_BOARDCOMAND (JOIN_SEQ, BC_TYPE, USR_SEQ, NICKNAME, CONTENTS, OPEN_YN, REG_DATE)
		VALUES (?, 'B', ?, ?, ?, 'Y', ?)
	`, joinSeq, usrSeq, regName, contents, time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// SoftDeleteComment sets OPEN_YN='N' for a comment owned by the given user.
func (r *CommentRepository) SoftDeleteComment(bcSeq int, usrSeq int) (int64, error) {
	result, err := r.DB.Exec(`
		UPDATE WEO_BOARDCOMAND SET OPEN_YN = 'N'
		WHERE SEQ = ? AND USR_SEQ = ?
	`, bcSeq, usrSeq)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
