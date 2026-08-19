package repository_test

import (
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TestCreateDonationOrderPersistsCanonicalAndLegacyFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock"))
	accountUsrSeq := 42
	order := model.NormalizedDonationOrder{
		DonationOrderInput: model.DonationOrderInput{
			Source:         "bank_transfer",
			AccountUsrSeq:  &accountUsrSeq,
			DonationDate:   "2026-07-28",
			Donor:          model.DonationDonor{Name: "예시 동문", Cohort: "18", Department: "영어", Phone: "01000000000"},
			DonationType:   "one_time",
			GrossAmount:    100000,
			RefundedAmount: 20000,
			Status:         "partially_refunded",
			PaymentMethod:  "bank",
		},
		NetReceivedAmount: 80000,
		CompositeKey:      "composite-key",
		LegacyGate:        "S",
		LegacyStatus:      "Y",
		LegacyPayment:     "Y",
	}

	mock.ExpectExec(`INSERT INTO WEO_ORDER \(\s*USR_SEQ, O_ACCOUNT_USR_SEQ`).
		WithArgs(
			0, accountUsrSeq, "bank_transfer", nil, "composite-key", "2026-07-28",
			"예시 동문", "01000000000", "18", "영어", "S",
			int64(100000), int64(20000), int64(80000), "partially_refunded", "bank", nil,
			int64(100000), int64(80000), "BANK", "Y", "Y", 7, "192.0.2.1", 7, "192.0.2.1",
		).
		WillReturnResult(sqlmock.NewResult(3001, 1))

	seq, err := repo.CreateDonationOrder(order, 7, "192.0.2.1")
	if err != nil {
		t.Fatalf("CreateDonationOrder() error = %v", err)
	}
	if seq != 3001 {
		t.Fatalf("seq = %d, want 3001", seq)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDonationOrderAllowsUnlinkedAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectExec(`INSERT INTO WEO_ORDER`).
		WillReturnResult(sqlmock.NewResult(3002, 1))

	seq, err := repo.CreateDonationOrder(model.NormalizedDonationOrder{}, 7, "192.0.2.1")
	if err != nil {
		t.Fatalf("CreateDonationOrder() error = %v", err)
	}
	if seq != 3002 {
		t.Fatalf("seq = %d, want 3002", seq)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDonationOrderClassifiesDuplicateIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectExec(`INSERT INTO WEO_ORDER`).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate"})

	_, err = repo.CreateDonationOrder(model.NormalizedDonationOrder{}, 7, "192.0.2.1")
	if !errors.Is(err, repository.ErrDonationOrderConflict) {
		t.Fatalf("error = %v, want ErrDonationOrderConflict", err)
	}
}

func TestGetDonationOrderReturnsCanonicalDetail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectQuery(`FROM WEO_ORDER o`).
		WithArgs(3001).
		WillReturnRows(sqlmock.NewRows([]string{
			"ORDER_SEQ", "SOURCE", "TRANSACTION_NUMBER", "DONATION_DATE",
			"DONOR_NAME", "DONOR_COHORT", "DONOR_DEPARTMENT", "DONOR_PHONE", "DONATION_TYPE",
			"GROSS_AMOUNT", "REFUNDED_AMOUNT", "NET_RECEIVED_AMOUNT", "STATUS", "PAYMENT_METHOD",
			"MEMO", "LAST_EDITED_BY", "LAST_EDITED_AT", "LAST_EDITED_IP",
		}).AddRow(
			3001, "bank_transfer", nil, "2026-07-28",
			"예시 동문", "18", "영어", "01000000000", "one_time",
			int64(100000), int64(20000), int64(80000), "partially_refunded", "bank",
			nil, 7, "2026-07-28T01:00:00Z", "192.0.2.1",
		))

	order, err := repo.GetDonationOrder(3001)
	if err != nil {
		t.Fatalf("GetDonationOrder() error = %v", err)
	}
	if order.OrderSeq != 3001 || order.Donor.Phone != "01000000000" || order.NetReceivedAmount != 80000 {
		t.Fatalf("unexpected order: %+v", order)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateDonationOrderChangesLinkedAccountWhenProvided(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock"))
	accountUsrSeq := 42
	order := model.NormalizedDonationOrder{
		DonationOrderInput: model.DonationOrderInput{
			Source: "other", AccountUsrSeq: &accountUsrSeq, AccountUsrSeqSet: true, DonationDate: "2026-07-29",
			Donor:        model.DonationDonor{Name: "기부자", Cohort: "19", Department: "중국어", Phone: "01011112222"},
			DonationType: "sponsorship", GrossAmount: 50000, Status: "completed", PaymentMethod: "admin",
		},
		NetReceivedAmount: 50000, CompositeKey: "replacement-key", LegacyGate: "F", LegacyStatus: "Y", LegacyPayment: "Y",
	}
	mock.ExpectExec(`O_ACCOUNT_USR_SEQ = CASE WHEN \? THEN \? ELSE O_ACCOUNT_USR_SEQ END`).
		WithArgs(
			true, accountUsrSeq, "other", nil, "replacement-key", "2026-07-29", "기부자", "01011112222", "19", "중국어", "F",
			int64(50000), int64(0), int64(50000), "completed", "admin", nil,
			int64(50000), int64(50000), "ADMS", "Y", "Y", 7, "192.0.2.1", int64(3001),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateDonationOrder(3001, order, 7, "192.0.2.1"); err != nil {
		t.Fatalf("UpdateDonationOrder() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateDonationOrderPreservesLinkedAccountWhenOmitted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock"))
	order := model.NormalizedDonationOrder{
		DonationOrderInput: model.DonationOrderInput{
			Source: "bank_transfer", DonationDate: "2026-07-28",
			Donor:        model.DonationDonor{Name: "예시 동문", Cohort: "18", Department: "영어", Phone: "01000000000"},
			DonationType: "one_time", GrossAmount: 100000, Status: "completed", PaymentMethod: "bank",
		},
		NetReceivedAmount: 100000, CompositeKey: "composite", LegacyGate: "S", LegacyStatus: "Y", LegacyPayment: "Y",
	}
	mock.ExpectExec(`O_ACCOUNT_USR_SEQ = CASE WHEN \? THEN \? ELSE O_ACCOUNT_USR_SEQ END`).
		WithArgs(
			false, nil, "bank_transfer", nil, "composite", "2026-07-28", "예시 동문", "01000000000", "18", "영어", "S",
			int64(100000), int64(0), int64(100000), "completed", "bank", nil,
			int64(100000), int64(100000), "BANK", "Y", "Y", 7, "192.0.2.1", int64(3001),
		).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.UpdateDonationOrder(3001, order, 7, "192.0.2.1"); err != nil {
		t.Fatalf("UpdateDonationOrder() unchanged replacement error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateDonationOrderClearsLinkedAccountWhenNullProvided(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock"))
	order := model.NormalizedDonationOrder{
		DonationOrderInput: model.DonationOrderInput{
			Source: "bank_transfer", AccountUsrSeqSet: true, DonationDate: "2026-07-28",
			Donor:        model.DonationDonor{Name: "예시 동문", Cohort: "18", Department: "영어", Phone: "01000000000"},
			DonationType: "one_time", GrossAmount: 100000, Status: "completed", PaymentMethod: "bank",
		},
		NetReceivedAmount: 100000, CompositeKey: "composite", LegacyGate: "S", LegacyStatus: "Y", LegacyPayment: "Y",
	}
	mock.ExpectExec(`O_ACCOUNT_USR_SEQ = CASE WHEN \? THEN \? ELSE O_ACCOUNT_USR_SEQ END`).
		WithArgs(
			true, nil, "bank_transfer", nil, "composite", "2026-07-28", "예시 동문", "01000000000", "18", "영어", "S",
			int64(100000), int64(0), int64(100000), "completed", "bank", nil,
			int64(100000), int64(100000), "BANK", "Y", "Y", 7, "192.0.2.1", int64(3001),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateDonationOrder(3001, order, 7, "192.0.2.1"); err != nil {
		t.Fatalf("UpdateDonationOrder() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListDonationOrdersAppliesCanonicalFiltersAndStablePagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock"))
	filterArgs := []driver.Value{"%동문%", "%0100000%", "%TX%", "bank_transfer", "completed", "S"}
	mock.ExpectQuery(`SELECT COUNT\(\*\).*FROM WEO_ORDER o`).
		WithArgs(filterArgs...).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(21))
	listArgs := append(append([]driver.Value{}, filterArgs...), 20, 20)
	mock.ExpectQuery(`ORDER BY o.O_DONATION_DATE DESC, o.O_SEQ DESC`).
		WithArgs(listArgs...).
		WillReturnRows(sqlmock.NewRows([]string{
			"ORDER_SEQ", "SOURCE", "TRANSACTION_NUMBER", "DONATION_DATE",
			"DONOR_NAME", "DONOR_COHORT", "DONOR_DEPARTMENT", "DONOR_PHONE", "DONATION_TYPE",
			"GROSS_AMOUNT", "REFUNDED_AMOUNT", "NET_RECEIVED_AMOUNT", "STATUS", "PAYMENT_METHOD",
			"MEMO", "LAST_EDITED_BY", "LAST_EDITED_AT", "LAST_EDITED_IP",
		}).AddRow(
			3001, "bank_transfer", "TX-1", "2026-07-28", "예시 동문", "18", "영어", "01000000000", "one_time",
			int64(100000), int64(0), int64(100000), "completed", "bank", nil, 7, "2026-07-28T01:00:00Z", "192.0.2.1",
		))

	orders, total, err := repo.ListDonationOrders(model.DonationOrderFilters{
		Name: "동문", Phone: "0100000", TransactionNumber: "TX", Source: "bank_transfer", Status: "completed", DonationType: "one_time",
	}, 2, 20)
	if err != nil {
		t.Fatalf("ListDonationOrders() error = %v", err)
	}
	if total != 21 || len(orders) != 1 || orders[0].OrderSeq != 3001 {
		t.Fatalf("orders/total = %+v/%d", orders, total)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
