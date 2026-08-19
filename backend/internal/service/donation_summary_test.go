package service_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
)

func TestDonationSummaryUsesSnapshotCalculationAndCacheInvalidation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
	cacheStore := cache.New(5*time.Minute, 10*time.Minute)
	donationService := service.NewDonationService(repo, cacheStore)

	expectSnapshot(mock, "2026-08-20", 180000, 20000, 12, 500000, "N")
	mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(donationConfigRows("N", 0))
	first, err := donationService.GetSummary()
	if err != nil {
		t.Fatalf("GetSummary() error = %v", err)
	}
	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"displayAmount":200000,"goalAmount":500000,"donorCount":12,"achievementRate":40,"snapshotDate":"2026-08-20","tierThresholds":{"sprout":1,"sapling":10000,"tree":50000,"blooming":100000,"fruiting":300000}}`
	if string(encoded) != want {
		t.Fatalf("summary JSON = %s, want %s", encoded, want)
	}
	if _, err := donationService.GetSummary(); err != nil {
		t.Fatalf("cached GetSummary() error = %v", err)
	}

	donationService.InvalidateCache()
	expectSnapshot(mock, "2026-08-20", 200000, 20000, 13, 500000, "N")
	mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(donationConfigRows("N", 0))
	refreshed, err := donationService.GetSummary()
	if err != nil {
		t.Fatalf("refreshed GetSummary() error = %v", err)
	}
	if refreshed.DisplayAmount != 220000 || refreshed.DonorCount != 13 {
		t.Fatalf("refreshed summary = %+v", refreshed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDonationSummaryFallsBackToLiveCalculationWithManualOverwrite(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
	donationService := service.NewDonationService(repo, cache.New(5*time.Minute, 10*time.Minute))

	mock.ExpectQuery(`(?s)FROM DONATION_SNAPSHOT.*WHERE DS_DATE`).WillReturnRows(donationSnapshotRows())
	mock.ExpectQuery(`(?s)FROM DONATION_SNAPSHOT.*ORDER BY DS_DATE DESC`).WillReturnRows(donationSnapshotRows())
	mock.ExpectQuery(`(?s)SUM\(O_PRICE\).*O_PAYMENT = 'Y'`).WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(int64(180000)))
	mock.ExpectQuery(`COUNT\(DISTINCT USR_SEQ\)`).WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(12))
	mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(donationConfigRows("Y", 25))

	summary, err := donationService.GetSummary()
	if err != nil {
		t.Fatalf("GetSummary() error = %v", err)
	}
	if summary.DisplayAmount != 20000 || summary.DonorCount != 25 || summary.AchievementRate != 4 {
		t.Fatalf("summary = %+v", summary)
	}
	if summary.SnapshotDate != time.Now().Format("2006-01-02") {
		t.Fatalf("snapshotDate = %q", summary.SnapshotDate)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectSnapshot(mock sqlmock.Sqlmock, date string, total, manualAdj int64, donorCount int, goal int64, overwrite string) {
	mock.ExpectQuery(`(?s)FROM DONATION_SNAPSHOT.*WHERE DS_DATE`).
		WillReturnRows(donationSnapshotRows().AddRow(1, date, total, manualAdj, donorCount, goal, overwrite, "2026-08-20 00:05:00"))
}

func donationSnapshotRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"DS_SEQ", "DS_DATE", "DS_TOTAL", "DS_MANUAL_ADJ", "DS_DONOR_CNT", "DS_GOAL", "DS_OVERWRITE", "REG_DATE",
	})
}

func donationConfigRows(overwrite string, manualDonorCount int) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"DC_GOAL", "DC_MANUAL_ADJ", "DC_MANUAL_DONOR_CNT",
		"DC_TIER_SPROUT_MIN", "DC_TIER_SAPLING_MIN", "DC_TIER_TREE_MIN",
		"DC_TIER_BLOOMING_MIN", "DC_TIER_FRUITING_MIN", "DC_OVERWRITE",
	}).AddRow(
		int64(500000), int64(20000), manualDonorCount,
		int64(1), int64(10000), int64(50000), int64(100000), int64(300000), overwrite,
	)
}
