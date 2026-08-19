package repository_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestGetActiveConfigReturnsDefaultDonationTierThresholds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`(?s)SELECT DC_SEQ, DC_GOAL.*DC_TIER_SPROUT_MIN.*DC_TIER_FRUITING_MIN.*FROM DONATION_CONFIG`).
		WillReturnRows(sqlmock.NewRows([]string{
			"DC_SEQ", "DC_GOAL", "DC_MANUAL_ADJ", "DC_MANUAL_DONOR_CNT",
			"DC_TIER_SPROUT_MIN", "DC_TIER_SAPLING_MIN", "DC_TIER_TREE_MIN",
			"DC_TIER_BLOOMING_MIN", "DC_TIER_FRUITING_MIN", "DC_NOTE",
			"DC_OVERWRITE", "IS_ACTIVE", "REG_DATE", "REG_OPER",
		}).AddRow(
			1, int64(200000000), int64(0), 0,
			int64(1), int64(10000), int64(50000), int64(100000), int64(300000), "초기 설정",
			"N", "Y", "2026-08-20 10:00:00", 0,
		))

	config, err := repo.GetActiveConfig()
	if err != nil {
		t.Fatalf("GetActiveConfig() error = %v", err)
	}
	if config == nil {
		t.Fatal("GetActiveConfig() = nil")
	}
	if config.TierSproutMin != 1 || config.TierSaplingMin != 10000 || config.TierTreeMin != 50000 ||
		config.TierBloomingMin != 100000 || config.TierFruitingMin != 300000 {
		t.Fatalf("tier thresholds = %d/%d/%d/%d/%d, want 1/10000/50000/100000/300000",
			config.TierSproutMin, config.TierSaplingMin, config.TierTreeMin,
			config.TierBloomingMin, config.TierFruitingMin)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateConfigPersistsDonationTierThresholds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock"))
	config := model.DonationConfig{
		Goal: 250000000, ManualAdj: 5000, ManualDonorCnt: 12,
		TierSproutMin: 10, TierSaplingMin: 20000, TierTreeMin: 60000,
		TierBloomingMin: 120000, TierFruitingMin: 350000,
		Note: "tier update", Overwrite: "Y",
	}

	mock.ExpectExec(`(?s)UPDATE DONATION_CONFIG.*DC_TIER_SPROUT_MIN = \?.*DC_TIER_FRUITING_MIN = \?.*WHERE IS_ACTIVE = 'Y'`).
		WithArgs(
			config.Goal, config.ManualAdj, config.ManualDonorCnt,
			config.TierSproutMin, config.TierSaplingMin, config.TierTreeMin,
			config.TierBloomingMin, config.TierFruitingMin,
			config.Note, config.Overwrite, 7,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateConfig(config, 7); err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
