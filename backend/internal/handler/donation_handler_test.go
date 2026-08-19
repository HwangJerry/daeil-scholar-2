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

func TestDonationSummaryExposesGoalAndTierThresholds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
	donationService := service.NewDonationService(repo, cache.New(5*time.Minute, 10*time.Minute))
	handler := NewDonationHandler(donationService)

	mock.ExpectQuery(`SUM\(O_NET_RECEIVED_AMOUNT\)`).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(int64(180000)))
	mock.ExpectQuery(`FROM DONATION_CONFIG`).
		WillReturnRows(sqlmock.NewRows([]string{
			"DC_GOAL", "DC_TIER_SPROUT_MIN", "DC_TIER_SAPLING_MIN", "DC_TIER_TREE_MIN",
			"DC_TIER_BLOOMING_MIN", "DC_TIER_FRUITING_MIN",
		}).AddRow(int64(200000000), int64(1), int64(10000), int64(50000), int64(100000), int64(300000)))
	request := httptest.NewRequest(http.MethodGet, "/api/donation/summary", nil)
	response := httptest.NewRecorder()

	handler.GetSummary(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	want := `{"displayAmount":180000,"goalAmount":200000000,"tierThresholds":{"sprout":1,"sapling":10000,"tree":50000,"blooming":100000,"fruiting":300000}}` + "\n"
	if response.Body.String() != want {
		t.Fatalf("body = %s, want %s", response.Body.String(), want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
