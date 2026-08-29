package repository

import (
	"database/sql"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type BannerAdRepository struct {
	DB *sqlx.DB
}

func NewBannerAdRepository(db *sqlx.DB) *BannerAdRepository {
	return &BannerAdRepository{DB: db}
}

func (r *BannerAdRepository) GetActiveBanner() (*model.BannerAd, error) {
	var banner model.BannerAd
	err := r.DB.Get(&banner, `
		SELECT BN_SEQ, BN_NAME, BN_URL,
		       DATE_FORMAT(BN_START_DATE, '%Y-%m-%dT%H:%i:%sZ') AS BN_START_DATE,
		       DATE_FORMAT(BN_END_DATE, '%Y-%m-%dT%H:%i:%sZ') AS BN_END_DATE
		FROM MAIN_BANNER_AD
		WHERE OPEN_YN = 'Y'
		  AND (BN_START_DATE IS NULL OR BN_START_DATE <= UTC_TIMESTAMP())
		  AND (BN_END_DATE IS NULL OR BN_END_DATE >= UTC_TIMESTAMP())
		ORDER BY INDX ASC
		LIMIT 1
	`)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	images := make([]model.BannerAdImage, 0)
	if err := r.DB.Select(&images, `
		SELECT BNI_SEQ, BN_SEQ, IMAGE_URL, SORT_ORDER
		FROM MAIN_BANNER_AD_IMAGE
		WHERE BN_SEQ = ?
		ORDER BY SORT_ORDER ASC
	`, banner.BNSeq); err != nil {
		return nil, err
	}
	banner.Images = images
	return &banner, nil
}

func (r *BannerAdRepository) LogEvent(bnSeq int, eventType string) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_BANNER_AD_LOG (BN_SEQ, LOG_TYPE, CREATED_AT)
		VALUES (?, ?, NOW())
	`, bnSeq, eventType)
	return err
}
