package repository_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestGetReceivedDonationAggregateUsesCanonicalNetLedgerAndDonorIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*COUNT\(DISTINCT CASE.*O_ACCOUNT_USR_SEQ IS NOT NULL.*O_DONOR_NAME.*O_DONOR_PHONE.*FROM WEO_ORDER.*O_TYPE = 'A'.*O_LIFECYCLE_STATUS IN \('completed', 'partially_refunded'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL_AMOUNT", "DONOR_COUNT"}).AddRow(int64(180000), 3))

	total, donorCount, err := repo.GetReceivedDonationAggregate()
	if err != nil {
		t.Fatalf("GetReceivedDonationAggregate() error = %v", err)
	}
	if total != 180000 {
		t.Fatalf("total = %d, want 180000", total)
	}
	if donorCount != 3 {
		t.Fatalf("donorCount = %d, want 3", donorCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetReceivedDonationAggregateReturnsZeroForEmptyLedger(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*O_TYPE = 'A'.*O_LIFECYCLE_STATUS IN \('completed', 'partially_refunded'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL_AMOUNT", "DONOR_COUNT"}).AddRow(int64(0), 0))

	total, donorCount, err := repo.GetReceivedDonationAggregate()
	if err != nil {
		t.Fatalf("GetReceivedDonationAggregate() error = %v", err)
	}
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
	if donorCount != 0 {
		t.Fatalf("donorCount = %d, want 0", donorCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
