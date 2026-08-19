package repository_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestEasyPayOrderWriterPersistsCanonicalPendingDonationFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonateRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`(?s)INSERT INTO WEO_ORDER \(.*O_ACCOUNT_USR_SEQ.*O_SOURCE.*O_DONATION_DATE.*O_GROSS_AMOUNT.*O_NET_RECEIVED_AMOUNT.*O_LIFECYCLE_STATUS.*O_PAYMENT_METHOD.*\).*VALUES .*'pending'`).
		WillReturnResult(sqlmock.NewResult(41, 1))

	seq, err := repo.InsertOrder(7, "S", "CARD", 50000, "192.0.2.1")
	if err != nil || seq != 41 {
		t.Fatalf("InsertOrder() = %d, %v", seq, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEasyPayApprovalWriterCompletesCanonicalDonationFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonateRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`(?s)UPDATE WEO_ORDER.*O_NET_RECEIVED_AMOUNT = \?.*O_LIFECYCLE_STATUS = 'completed'.*WHERE O_SEQ = \? AND O_PAYMENT = 'N'`).
		WithArgs(50000, int64(91), 50000, "192.0.2.1", 41).
		WillReturnResult(sqlmock.NewResult(0, 1))

	affected, err := repo.UpdateOrderPayment(41, 50000, 91, "192.0.2.1")
	if err != nil || affected != 1 {
		t.Fatalf("UpdateOrderPayment() = %d, %v", affected, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
