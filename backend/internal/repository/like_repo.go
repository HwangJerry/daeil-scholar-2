// like_repo.go — Repository for WEO_BOARDLIKE table operations
package repository

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// LikeRepository handles like-related database operations.
type LikeRepository struct {
	DB *sqlx.DB
}

// NewLikeRepository creates a new LikeRepository.
func NewLikeRepository(db *sqlx.DB) *LikeRepository {
	return &LikeRepository{DB: db}
}

// HasUserLiked checks if a user has an active like on a post.
func (r *LikeRepository) HasUserLiked(bbsSeq int, usrSeq int) (bool, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM WEO_BOARDLIKE
		WHERE BBS_SEQ = ? AND USR_SEQ = ? AND OPEN_YN = 'Y'
	`, bbsSeq, usrSeq)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindLike returns the SEQ and current OPEN_YN of an existing like row, or 0 if not found.
func (r *LikeRepository) FindLike(bbsSeq int, usrSeq int) (int, string, error) {
	var row struct {
		SEQ    int    `db:"SEQ"`
		OpenYN string `db:"OPEN_YN"`
	}
	err := r.DB.Get(&row, `
		SELECT SEQ, OPEN_YN FROM WEO_BOARDLIKE
		WHERE BBS_SEQ = ? AND USR_SEQ = ?
		LIMIT 1
	`, bbsSeq, usrSeq)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", nil
		}
		return 0, "", err
	}
	return row.SEQ, row.OpenYN, nil
}

// InsertLike creates a new like row with OPEN_YN='Y'.
func (r *LikeRepository) InsertLike(bbsSeq int, usrSeq int) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_BOARDLIKE (BBS_SEQ, USR_SEQ, OPEN_YN, REG_DATE)
		VALUES (?, ?, 'Y', ?)
	`, bbsSeq, usrSeq, time.Now().Format("2006-01-02 15:04:05"))
	return err
}

// SetLikeOpen updates the OPEN_YN flag for a like row.
func (r *LikeRepository) SetLikeOpen(seq int, openYN string) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_BOARDLIKE SET OPEN_YN = ? WHERE SEQ = ?
	`, openYN, seq)
	return err
}

// SetLikeOpenByUser updates OPEN_YN for ALL like rows matching a user+post pair.
// This handles duplicate rows that may exist for the same (BBS_SEQ, USR_SEQ).
func (r *LikeRepository) SetLikeOpenByUser(bbsSeq int, usrSeq int, openYN string) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_BOARDLIKE SET OPEN_YN = ? WHERE BBS_SEQ = ? AND USR_SEQ = ?
	`, openYN, bbsSeq, usrSeq)
	return err
}
