package repository_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

const personalDonationFilterPattern = `(?s)WHERE O_ACCOUNT_USR_SEQ = \?.*AND O_TYPE = 'A'.*AND O_LIFECYCLE_STATUS IN \('completed', 'partially_refunded'\)`

func TestPersonalDonationRepositoryGetTotalsUsesCanonicalAccountAndReceivedFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewPersonalDonationRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*` + personalDonationFilterPattern).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL_AMOUNT", "TOTAL_COUNT"}).AddRow(int64(180000), 2))

	totalAmount, totalCount, err := repo.GetTotals(42)
	if err != nil {
		t.Fatalf("GetTotals() error = %v", err)
	}
	if totalAmount != 180000 || totalCount != 2 {
		t.Fatalf("totals = %d/%d, want 180000/2", totalAmount, totalCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPersonalDonationRepositoryListUsesCanonicalFieldsAndPagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewPersonalDonationRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`(?s)O_SEQ AS ORDER_SEQ.*O_GROSS_AMOUNT.*O_REFUNDED_AMOUNT.*O_NET_RECEIVED_AMOUNT.*`+personalDonationFilterPattern+`.*ORDER BY O_DONATION_DATE DESC, O_SEQ DESC.*LIMIT \? OFFSET \?`).
		WithArgs(42, 2, 2).
		WillReturnRows(sqlmock.NewRows([]string{
			"ORDER_SEQ", "DONATION_DATE", "GROSS_AMOUNT", "REFUNDED_AMOUNT",
			"NET_RECEIVED_AMOUNT", "LIFECYCLE_STATUS", "PAYMENT_METHOD", "SOURCE",
		}).AddRow(3001, "2026-07-28", int64(100000), int64(20000), int64(80000), "partially_refunded", "bank", "bank_transfer"))

	items, err := repo.List(42, "latest", 2, 2)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 || items[0].OrderSeq != 3001 || items[0].NetReceivedAmount != 80000 {
		t.Fatalf("unexpected items: %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPersonalDonationRepositoryAmountSortUsesCanonicalNetAmount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewPersonalDonationRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`(?s)`+personalDonationFilterPattern+`.*ORDER BY O_NET_RECEIVED_AMOUNT DESC, O_SEQ DESC`).
		WithArgs(42, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"ORDER_SEQ", "DONATION_DATE", "GROSS_AMOUNT", "REFUNDED_AMOUNT",
			"NET_RECEIVED_AMOUNT", "LIFECYCLE_STATUS", "PAYMENT_METHOD", "SOURCE",
		}))

	items, err := repo.List(42, "amount", 1, 20)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("items = %#v, want non-nil empty slice", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
