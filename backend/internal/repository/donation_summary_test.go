package repository_test

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestReceivedDonationAmountBetweenUsesHalfOpenCanonicalDateRange(t *testing.T) {
	for _, tc := range []struct {
		name   string
		amount int64
		err    error
	}{
		{name: "received net amount", amount: 5000000000},
		{name: "empty month", amount: 0},
		{name: "database error", err: errors.New("ledger unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
			query := mock.ExpectQuery(`(?s)CAST\(COALESCE\(SUM\(O_NET_RECEIVED_AMOUNT\), 0\) AS SIGNED\).*FROM WEO_ORDER.*O_TYPE = 'A'.*O_LIFECYCLE_STATUS IN \('completed', 'partially_refunded'\).*O_DONATION_DATE >= \? AND O_DONATION_DATE < \?`).
				WithArgs("2024-02-01", "2024-03-01")
			if tc.err != nil {
				query.WillReturnError(tc.err)
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"AMOUNT"}).AddRow(tc.amount))
			}
			start := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
			amount, err := repo.GetReceivedDonationAmountBetween(start, start.AddDate(0, 1, 0))
			if !errors.Is(err, tc.err) || amount != tc.amount {
				t.Fatalf("amount=%d, error=%v; want %d/%v", amount, err, tc.amount, tc.err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetReceivedDonationAggregateUsesCanonicalNetLedgerAndDonorIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*COUNT\(DISTINCT CASE.*O_ACCOUNT_USR_SEQ IS NOT NULL.*O_DONOR_NAME.*O_DONOR_PHONE.*FROM WEO_ORDER.*O_TYPE = 'A'.*O_LIFECYCLE_STATUS IN \('completed', 'partially_refunded'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL_AMOUNT", "DONOR_COUNT"}).AddRow(int64(180000), 3))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema.TABLES`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

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
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema.TABLES`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

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
