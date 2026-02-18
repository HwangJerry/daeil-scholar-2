package repository

import (
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AdRepository struct {
	DB *sqlx.DB
}

func NewAdRepository(db *sqlx.DB) *AdRepository {
	return &AdRepository{DB: db}
}

func (r *AdRepository) GetActiveAds(excludeIDs []int) ([]model.AdItem, error) {
	args := []interface{}{}
	query := strings.Builder{}
	query.WriteString(`
		SELECT MA_SEQ, IFNULL(MA_NAME,'') AS MA_NAME, IFNULL(MA_URL,'') AS MA_URL,
		       IFNULL(MA_IMG,'') AS MA_IMG, AD_TIER, AD_TITLE_LABEL
		FROM MAIN_AD
		WHERE OPEN_YN = 'Y'
	`)
	if len(excludeIDs) > 0 {
		placeholders := make([]string, len(excludeIDs))
		for i, id := range excludeIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		query.WriteString(" AND MA_SEQ NOT IN (")
		query.WriteString(strings.Join(placeholders, ","))
		query.WriteString(")")
	}
	query.WriteString(" ORDER BY FIELD(AD_TIER, 'PREMIUM','GOLD','NORMAL'), INDX ASC")
	var ads []model.AdItem
	if err := r.DB.Select(&ads, query.String(), args...); err != nil {
		return nil, err
	}
	return ads, nil
}

func (r *AdRepository) LogAdEvent(maSeq int, usrSeq int, eventType string, ipAddr string) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_AD_LOG (MA_SEQ, USR_SEQ, AL_TYPE, AL_DATE, AL_IPADDR)
		VALUES (?, NULLIF(?, 0), ?, NOW(), ?)
	`, maSeq, usrSeq, eventType, ipAddr)
	return err
}
