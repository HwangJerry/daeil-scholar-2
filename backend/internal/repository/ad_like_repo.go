// ad_like_repo.go — Repository for WEO_AD_LIKE table operations
package repository

import (
	"time"

	"github.com/jmoiron/sqlx"
)

// AdLikeRepository handles WEO_AD_LIKE database operations.
type AdLikeRepository struct {
	DB *sqlx.DB
}

// NewAdLikeRepository creates a new AdLikeRepository.
func NewAdLikeRepository(db *sqlx.DB) *AdLikeRepository {
	return &AdLikeRepository{DB: db}
}

// HasUserLikedAd checks if a user has an active like on an ad.
func (r *AdLikeRepository) HasUserLikedAd(maSeq, usrSeq int) (bool, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM WEO_AD_LIKE WHERE MA_SEQ = ? AND USR_SEQ = ? AND OPEN_YN = 'Y'
	`, maSeq, usrSeq)
	return count > 0, err
}

// HasAnyAdLikeRow returns true if any row (active or inactive) exists for this user+ad.
func (r *AdLikeRepository) HasAnyAdLikeRow(maSeq, usrSeq int) (bool, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM WEO_AD_LIKE WHERE MA_SEQ = ? AND USR_SEQ = ?
	`, maSeq, usrSeq)
	return count > 0, err
}

// InsertAdLike creates a new ad like row with OPEN_YN='Y'.
func (r *AdLikeRepository) InsertAdLike(maSeq, usrSeq int) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_AD_LIKE (MA_SEQ, USR_SEQ, OPEN_YN, REG_DATE) VALUES (?, ?, 'Y', ?)
	`, maSeq, usrSeq, time.Now().Format("2006-01-02 15:04:05"))
	return err
}

// SetAdLikeOpenByUser updates OPEN_YN for all like rows matching a user+ad pair.
func (r *AdLikeRepository) SetAdLikeOpenByUser(maSeq, usrSeq int, openYN string) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_AD_LIKE SET OPEN_YN = ? WHERE MA_SEQ = ? AND USR_SEQ = ?
	`, openYN, maSeq, usrSeq)
	return err
}

// GetAdLikeCount returns the number of active likes for an ad.
func (r *AdLikeRepository) GetAdLikeCount(maSeq int) (int, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM WEO_AD_LIKE WHERE MA_SEQ = ? AND OPEN_YN = 'Y'
	`, maSeq)
	return count, err
}
