// Admin ad repository — CRUD queries for MAIN_AD admin operations
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AdminAdRepository struct {
	DB *sqlx.DB
}

func NewAdminAdRepository(db *sqlx.DB) *AdminAdRepository {
	return &AdminAdRepository{DB: db}
}

func (r *AdminAdRepository) GetAds() ([]model.AdminAdRow, error) {
	var ads []model.AdminAdRow
	err := r.DB.Select(&ads, `
		SELECT MA_SEQ, IFNULL(MA_NAME,'') AS MA_NAME, IFNULL(MA_URL,'') AS MA_URL,
		       IFNULL(MA_IMG,'') AS MA_IMG, OPEN_YN, AD_TIER, AD_TITLE_LABEL, INDX,
		       IFNULL(AD_START_DATE,'') AS AD_START_DATE, IFNULL(AD_END_DATE,'') AS AD_END_DATE
		FROM MAIN_AD ORDER BY INDX ASC
	`)
	return ads, err
}

func (r *AdminAdRepository) InsertAd(a *model.AdminAdInsert) (int, error) {
	res, err := r.DB.Exec(`
		INSERT INTO MAIN_AD (MA_NAME, MA_URL, MA_IMG, OPEN_YN, AD_TIER, AD_TITLE_LABEL, INDX, MA_TYPE, AD_START_DATE, AD_END_DATE)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'FEED', ?, ?)
	`, a.MAName, a.MAURL, a.MAImg, a.MAStatus, a.ADTier, a.ADTitleLabel, a.MAIndx, a.ADStartDate, a.ADEndDate)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *AdminAdRepository) UpdateAd(seq int, a *model.AdminAdInsert) error {
	_, err := r.DB.Exec(`
		UPDATE MAIN_AD
		SET MA_NAME = ?, MA_URL = ?, MA_IMG = ?, OPEN_YN = ?, AD_TIER = ?, AD_TITLE_LABEL = ?, INDX = ?,
		    AD_START_DATE = ?, AD_END_DATE = ?
		WHERE MA_SEQ = ?
	`, a.MAName, a.MAURL, a.MAImg, a.MAStatus, a.ADTier, a.ADTitleLabel, a.MAIndx, a.ADStartDate, a.ADEndDate, seq)
	return err
}

func (r *AdminAdRepository) CountActiveTierAds(tier string, excludeSeq int) (int, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM MAIN_AD
		WHERE AD_TIER = ? AND OPEN_YN = 'Y' AND MA_SEQ != ?
	`, tier, excludeSeq)
	return count, err
}

func (r *AdminAdRepository) DeleteAd(seq int) error {
	_, err := r.DB.Exec(`DELETE FROM MAIN_AD WHERE MA_SEQ = ?`, seq)
	return err
}

func (r *AdminAdRepository) GetAdStats(maSeq int) (*model.AdminAdStats, error) {
	var stats model.AdminAdStats
	stats.MASeq = maSeq
	if err := r.DB.Get(&stats.ViewCount, `SELECT COUNT(*) FROM WEO_AD_LOG WHERE MA_SEQ = ? AND AL_TYPE = 'VIEW'`, maSeq); err != nil {
		return nil, err
	}
	if err := r.DB.Get(&stats.ClickCount, `SELECT COUNT(*) FROM WEO_AD_LOG WHERE MA_SEQ = ? AND AL_TYPE = 'CLICK'`, maSeq); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *AdminAdRepository) GetTotalImpressions() (int, error) {
	var c int
	err := r.DB.Get(&c, `SELECT COUNT(*) FROM WEO_AD_LOG WHERE AL_TYPE = 'VIEW'`)
	return c, err
}

func (r *AdminAdRepository) GetTotalClicks() (int, error) {
	var c int
	err := r.DB.Get(&c, `SELECT COUNT(*) FROM WEO_AD_LOG WHERE AL_TYPE = 'CLICK'`)
	return c, err
}
