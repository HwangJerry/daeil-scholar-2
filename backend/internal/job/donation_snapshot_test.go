package job

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

func TestDonationSnapshotUsesCanonicalReceivedDonationAggregate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
	job := NewDonationSnapshotJob(repo, zerolog.Nop())
	mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*COUNT\(DISTINCT CASE.*O_TYPE = 'A'.*O_LIFECYCLE_STATUS IN \('completed', 'partially_refunded'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL_AMOUNT", "DONOR_COUNT"}).AddRow(int64(75000), 2))
	mock.ExpectQuery(`FROM DONATION_CONFIG`).
		WillReturnRows(sqlmock.NewRows([]string{
			"DC_GOAL", "DC_MANUAL_ADJ", "DC_MANUAL_DONOR_CNT",
			"DC_TIER_SPROUT_MIN", "DC_TIER_SAPLING_MIN", "DC_TIER_TREE_MIN",
			"DC_TIER_BLOOMING_MIN", "DC_TIER_FRUITING_MIN", "DC_OVERWRITE",
		}).AddRow(int64(500000), int64(5000), 0, int64(1), int64(10000), int64(50000), int64(100000), int64(300000), "N"))
	mock.ExpectExec(`(?s)INSERT INTO DONATION_SNAPSHOT.*DS_TOTAL.*DS_DONOR_CNT.*ON DUPLICATE KEY UPDATE`).
		WithArgs(sqlmock.AnyArg(), int64(75000), int64(5000), 2, int64(500000), "N").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := job.CreateSnapshotNow(); err != nil {
		t.Fatalf("CreateSnapshotNow() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
