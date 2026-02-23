// ad_comment_repo.go — Repository for WEO_AD_COMMENT table operations
package repository

import (
	"fmt"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// AdCommentRepository handles WEO_AD_COMMENT database operations.
type AdCommentRepository struct {
	DB *sqlx.DB
}

// NewAdCommentRepository creates a new AdCommentRepository.
func NewAdCommentRepository(db *sqlx.DB) *AdCommentRepository {
	return &AdCommentRepository{DB: db}
}

// ListAdComments returns all active comments for an ad, ordered oldest-first.
func (r *AdCommentRepository) ListAdComments(maSeq int) ([]model.AdComment, error) {
	var comments []model.AdComment
	err := r.DB.Select(&comments, `
		SELECT AC_SEQ, MA_SEQ, USR_SEQ, NICKNAME, CONTENTS, REG_DATE
		FROM WEO_AD_COMMENT
		WHERE MA_SEQ = ? AND OPEN_YN = 'Y'
		ORDER BY AC_SEQ ASC
	`, maSeq)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

// InsertAdComment creates a new ad comment and returns the created row.
func (r *AdCommentRepository) InsertAdComment(maSeq, usrSeq int, nickname, contents string) (model.AdComment, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := r.DB.Exec(`
		INSERT INTO WEO_AD_COMMENT (MA_SEQ, USR_SEQ, NICKNAME, CONTENTS, OPEN_YN, REG_DATE)
		VALUES (?, ?, ?, ?, 'Y', ?)
	`, maSeq, usrSeq, nickname, contents, now)
	if err != nil {
		return model.AdComment{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.AdComment{}, err
	}
	return model.AdComment{
		ACSeq:    int(id),
		MASeq:    maSeq,
		USRSeq:   usrSeq,
		Nickname: nickname,
		Contents: contents,
		RegDate:  now,
	}, nil
}

// DeleteAdComment soft-deletes a comment by setting OPEN_YN='N'.
// Returns an error if the comment does not exist or does not belong to usrSeq.
func (r *AdCommentRepository) DeleteAdComment(acSeq, usrSeq int) error {
	res, err := r.DB.Exec(`
		UPDATE WEO_AD_COMMENT SET OPEN_YN = 'N' WHERE AC_SEQ = ? AND USR_SEQ = ?
	`, acSeq, usrSeq)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("댓글을 찾을 수 없거나 권한이 없습니다")
	}
	return nil
}
