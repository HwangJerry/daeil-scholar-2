package repository_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestSumNetReceivedDonationsIncludesOnlyReceivedStatuses(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*O_LIFECYCLE_STATUS IN \('completed', 'partially_refunded'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(int64(180000)))

	total, err := repo.SumNetReceivedDonations()
	if err != nil {
		t.Fatalf("SumNetReceivedDonations() error = %v", err)
	}
	if total != 180000 {
		t.Fatalf("total = %d, want 180000", total)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSumNetReceivedDonationsReturnsZeroForEmptyLedger(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*O_LIFECYCLE_STATUS IN \('completed', 'partially_refunded'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(int64(0)))

	total, err := repo.SumNetReceivedDonations()
	if err != nil {
		t.Fatalf("SumNetReceivedDonations() error = %v", err)
	}
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
