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

func TestDonationSummaryUsesCanonicalNetAggregateAndCacheInvalidation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
	cacheStore := cache.New(5*time.Minute, 10*time.Minute)
	donationService := service.NewDonationService(repo, cacheStore)

	query := `(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*O_LIFECYCLE_STATUS IN \('completed', 'partially_refunded'\)`
	mock.ExpectQuery(query).WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(int64(180000)))
	mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(donationConfigRows())
	first, err := donationService.GetSummary()
	if err != nil {
		t.Fatalf("GetSummary() error = %v", err)
	}
	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"displayAmount":180000,"goalAmount":200000000,"tierThresholds":{"sprout":1,"sapling":10000,"tree":50000,"blooming":100000,"fruiting":300000}}` {
		t.Fatalf("summary JSON = %s", encoded)
	}
	if _, err := donationService.GetSummary(); err != nil {
		t.Fatalf("cached GetSummary() error = %v", err)
	}

	donationService.InvalidateCache()
	mock.ExpectQuery(query).WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(int64(200000)))
	mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(donationConfigRows())
	refreshed, err := donationService.GetSummary()
	if err != nil {
		t.Fatalf("refreshed GetSummary() error = %v", err)
	}
	if refreshed.DisplayAmount != 200000 {
		t.Fatalf("refreshed displayAmount = %d", refreshed.DisplayAmount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func donationConfigRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"DC_GOAL", "DC_TIER_SPROUT_MIN", "DC_TIER_SAPLING_MIN", "DC_TIER_TREE_MIN",
		"DC_TIER_BLOOMING_MIN", "DC_TIER_FRUITING_MIN",
	}).AddRow(int64(200000000), int64(1), int64(10000), int64(50000), int64(100000), int64(300000))
}
