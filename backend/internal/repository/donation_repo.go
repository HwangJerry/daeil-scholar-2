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

const canonicalReceivedDonationPredicate = `
	O_TYPE = 'A'
	AND O_LIFECYCLE_STATUS IN ('completed', 'partially_refunded')`

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

func (r *DonationRepository) GetReceivedDonationAggregate() (int64, int, error) {
	var aggregate struct {
		TotalAmount int64 `db:"TOTAL_AMOUNT"`
		DonorCount  int   `db:"DONOR_COUNT"`
	}
	err := r.DB.Get(&aggregate, `
		SELECT
			CAST(COALESCE(SUM(O_NET_RECEIVED_AMOUNT), 0) AS SIGNED) AS TOTAL_AMOUNT,
			CAST(COUNT(DISTINCT CASE
				WHEN O_ACCOUNT_USR_SEQ IS NOT NULL AND O_ACCOUNT_USR_SEQ > 0
					THEN CONCAT('account:', O_ACCOUNT_USR_SEQ)
				WHEN COALESCE(TRIM(O_DONOR_NAME), '') <> '' OR COALESCE(TRIM(O_DONOR_PHONE), '') <> ''
					THEN CONCAT(
						'unlinked:', HEX(TRIM(COALESCE(O_DONOR_NAME, ''))), ':',
						REPLACE(TRIM(COALESCE(O_DONOR_PHONE, '')), '-', '')
					)
				ELSE CONCAT('order:', O_SEQ)
			END) AS SIGNED) AS DONOR_COUNT
		FROM WEO_ORDER
		WHERE `+canonicalReceivedDonationPredicate)
	if err != nil {
		return 0, 0, err
	}
	return aggregate.TotalAmount, aggregate.DonorCount, nil
}

func (r *DonationRepository) GetActiveConfig() (*model.DonationConfig, error) {
	var cfg model.DonationConfig
	err := r.DB.Get(&cfg, `
		SELECT DC_SEQ, DC_GOAL, DC_MANUAL_ADJ,
		       IFNULL(DC_MANUAL_DONOR_CNT,0) AS DC_MANUAL_DONOR_CNT,
		       DC_TIER_SPROUT_MIN, DC_TIER_SAPLING_MIN, DC_TIER_TREE_MIN,
		       DC_TIER_BLOOMING_MIN, DC_TIER_FRUITING_MIN,
		       IFNULL(DC_NOTE,'') AS DC_NOTE,
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
