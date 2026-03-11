// ad_repo.go — Repository for MAIN_AD table operations (retrieval and event logging)
package repository

import (
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// AdRepository handles MAIN_AD retrieval and event logging.
type AdRepository struct {
	DB *sqlx.DB
}

// NewAdRepository creates a new AdRepository.
func NewAdRepository(db *sqlx.DB) *AdRepository {
	return &AdRepository{DB: db}
}

// GetActiveAds returns all active ads, excluding any with IDs in excludeIDs.
// Engagement counts (likeCnt, commentCnt, hit) are fetched via correlated subqueries.
func (r *AdRepository) GetActiveAds(excludeIDs []int) ([]model.AdItem, error) {
	args := []interface{}{}
	query := strings.Builder{}
	query.WriteString(`
		SELECT a.MA_SEQ, IFNULL(a.MA_NAME,'') AS MA_NAME, IFNULL(a.MA_URL,'') AS MA_URL,
		       IFNULL(a.MA_IMG,'') AS MA_IMG, a.AD_TIER, a.AD_TITLE_LABEL,
		       (SELECT COUNT(*) FROM WEO_AD_LIKE    WHERE MA_SEQ = a.MA_SEQ AND OPEN_YN='Y') AS like_cnt,
		       (SELECT COUNT(*) FROM WEO_AD_COMMENT WHERE MA_SEQ = a.MA_SEQ AND OPEN_YN='Y') AS comment_cnt,
		       (SELECT COUNT(*) FROM WEO_AD_LOG     WHERE MA_SEQ = a.MA_SEQ AND AL_TYPE='VIEW') AS hit
		FROM MAIN_AD a
		WHERE a.OPEN_YN = 'Y'
	`)
	if len(excludeIDs) > 0 {
		placeholders := make([]string, len(excludeIDs))
		for i, id := range excludeIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		query.WriteString(" AND a.MA_SEQ NOT IN (")
		query.WriteString(strings.Join(placeholders, ","))
		query.WriteString(")")
	}
	query.WriteString(" ORDER BY FIELD(a.AD_TIER, 'PREMIUM','GOLD','NORMAL'), a.INDX ASC")
	var ads []model.AdItem
	if err := r.DB.Select(&ads, query.String(), args...); err != nil {
		return nil, err
	}
	return ads, nil
}

// GetUserLikedAdSeqs returns the maSeq values the user has liked from the given list.
func (r *AdRepository) GetUserLikedAdSeqs(userSeq int, maSeqs []int) ([]int, error) {
	if len(maSeqs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(maSeqs))
	args := make([]interface{}, 0, len(maSeqs)+1)
	args = append(args, userSeq)
	for i, seq := range maSeqs {
		placeholders[i] = "?"
		args = append(args, seq)
	}
	query := "SELECT MA_SEQ FROM WEO_AD_LIKE WHERE USR_SEQ = ? AND MA_SEQ IN (" +
		strings.Join(placeholders, ",") + ") AND OPEN_YN = 'Y'"
	var result []int
	if err := r.DB.Select(&result, query, args...); err != nil {
		return nil, err
	}
	return result, nil
}

// LogAdEvent records a view or click event for an ad.
func (r *AdRepository) LogAdEvent(maSeq int, usrSeq int, eventType string, ipAddr string) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_AD_LOG (MA_SEQ, USR_SEQ, AL_TYPE, AL_DATE, AL_IPADDR)
		VALUES (?, NULLIF(?, 0), ?, NOW(), ?)
	`, maSeq, usrSeq, eventType, ipAddr)
	return err
}
