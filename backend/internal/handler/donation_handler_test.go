package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
)

func TestDonationSummaryExposesRestoredSnapshotFieldsAndTierThresholds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
	donationService := service.NewDonationService(repo, cache.New(5*time.Minute, 10*time.Minute))
	handler := NewDonationHandler(donationService)

	mock.ExpectQuery(`(?s)FROM DONATION_SNAPSHOT.*WHERE DS_DATE`).
		WillReturnRows(sqlmock.NewRows([]string{
			"DS_SEQ", "DS_DATE", "DS_TOTAL", "DS_MANUAL_ADJ", "DS_DONOR_CNT", "DS_GOAL", "DS_OVERWRITE", "REG_DATE",
		}).AddRow(1, "2026-08-20", int64(180000), int64(20000), 12, int64(500000), "N", "2026-08-20 00:05:00"))
	mock.ExpectQuery(`FROM DONATION_CONFIG`).
		WillReturnRows(sqlmock.NewRows([]string{
			"DC_TIER_SPROUT_MIN", "DC_TIER_SAPLING_MIN", "DC_TIER_TREE_MIN",
			"DC_TIER_BLOOMING_MIN", "DC_TIER_FRUITING_MIN",
		}).AddRow(int64(1), int64(10000), int64(50000), int64(100000), int64(300000)))
	request := httptest.NewRequest(http.MethodGet, "/api/donation/summary", nil)
	response := httptest.NewRecorder()

	handler.GetSummary(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	want := `{"displayAmount":200000,"goalAmount":500000,"donorCount":12,"achievementRate":40,"snapshotDate":"2026-08-20","tierThresholds":{"sprout":1,"sapling":10000,"tree":50000,"blooming":100000,"fruiting":300000}}` + "\n"
	if response.Body.String() != want {
		t.Fatalf("body = %s, want %s", response.Body.String(), want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
