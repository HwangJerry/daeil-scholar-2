package repository

import (
	"database/sql"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type DonationRepository struct {
	DB *sqlx.DB
}

func NewDonationRepository(db *sqlx.DB) *DonationRepository {
	return &DonationRepository{DB: db}
}

func (r *DonationRepository) GetSnapshotByDate(date time.Time) (*model.DonationSnapshot, error) {
	var snapshot model.DonationSnapshot
	err := r.DB.Get(&snapshot, `
		SELECT DS_SEQ, DS_DATE, DS_TOTAL, DS_MANUAL_ADJ, DS_DONOR_CNT, DS_GOAL,
		       IFNULL(DS_OVERWRITE,'N') AS DS_OVERWRITE,
		       IFNULL(DATE_FORMAT(REG_DATE,'%Y-%m-%d %H:%i:%s'),'') AS REG_DATE
		FROM DONATION_SNAPSHOT
		WHERE DS_DATE = DATE(?)
		LIMIT 1
	`, date)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &snapshot, nil
}

func (r *DonationRepository) GetLatestSnapshot() (*model.DonationSnapshot, error) {
	var snapshot model.DonationSnapshot
	err := r.DB.Get(&snapshot, `
		SELECT DS_SEQ, DS_DATE, DS_TOTAL, DS_MANUAL_ADJ, DS_DONOR_CNT, DS_GOAL,
		       IFNULL(DS_OVERWRITE,'N') AS DS_OVERWRITE,
		       IFNULL(DATE_FORMAT(REG_DATE,'%Y-%m-%d %H:%i:%s'),'') AS REG_DATE
		FROM DONATION_SNAPSHOT
		ORDER BY DS_DATE DESC
		LIMIT 1
	`)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &snapshot, nil
}

func (r *DonationRepository) HasSnapshotForDate(date time.Time) (bool, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM DONATION_SNAPSHOT WHERE DS_DATE = DATE(?)
	`, date)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *DonationRepository) UpsertSnapshot(date time.Time, total int64, manualAdj int64, donorCnt int, goal int64, overwrite string) error {
	_, err := r.DB.Exec(`
		INSERT INTO DONATION_SNAPSHOT
			(DS_DATE, DS_TOTAL, DS_MANUAL_ADJ, DS_DONOR_CNT, DS_GOAL, DS_OVERWRITE, REG_DATE)
		VALUES (DATE(?), ?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
			DS_TOTAL = VALUES(DS_TOTAL),
			DS_MANUAL_ADJ = VALUES(DS_MANUAL_ADJ),
			DS_DONOR_CNT = VALUES(DS_DONOR_CNT),
			DS_GOAL = VALUES(DS_GOAL),
			DS_OVERWRITE = VALUES(DS_OVERWRITE)
	`, date, total, manualAdj, donorCnt, goal, overwrite)
	return err
}

func (r *DonationRepository) SumDonations() (int64, error) {
	var total int64
	err := r.DB.Get(&total, `
		SELECT CAST(COALESCE(SUM(O_PRICE), 0) AS SIGNED)
		FROM WEO_ORDER
		WHERE O_PAYMENT = 'Y'
	`)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (r *DonationRepository) CountDonors() (int, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT CAST(COUNT(DISTINCT USR_SEQ) AS SIGNED)
		FROM WEO_ORDER
		WHERE O_PAYMENT = 'Y'
	`)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *DonationRepository) GetActiveConfig() (*model.DonationConfig, error) {
	var cfg model.DonationConfig
	err := r.DB.Get(&cfg, `
		SELECT DC_SEQ, DC_GOAL, DC_MANUAL_ADJ, IFNULL(DC_NOTE,'') AS DC_NOTE,
		       IFNULL(DC_OVERWRITE,'N') AS DC_OVERWRITE, IS_ACTIVE,
		       IFNULL(DATE_FORMAT(REG_DATE,'%Y-%m-%d %H:%i:%s'),'') AS REG_DATE,
		       IFNULL(REG_OPER,0) AS REG_OPER
		FROM DONATION_CONFIG
		WHERE IS_ACTIVE = 'Y'
		ORDER BY DC_SEQ DESC
		LIMIT 1
	`)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}
