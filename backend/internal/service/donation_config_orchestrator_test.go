package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/job"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type snapshotCreatorStub struct {
	calls int
	err   error
}

func (s *snapshotCreatorStub) CreateSnapshotNow() error {
	s.calls++
	return s.err
}

func (s *snapshotCreatorStub) CreateSnapshotTx(_ *sqlx.Tx) error {
	s.calls++
	return s.err
}

func TestDonationOrderCreateRefreshesExistingSnapshotBeforeNextSummaryRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	donationRepo := repository.NewDonationRepository(sqlxDB)
	donationService := service.NewDonationService(donationRepo, cache.New(5*time.Minute, 10*time.Minute))
	adminService := service.NewAdminDonationService(repository.NewAdminDonationRepository(sqlxDB), donationRepo)
	snapshotCreator := &snapshotCreatorStub{}
	orchestrator := service.NewDonationConfigOrchestrator(adminService, donationService, snapshotCreator)

	expectSnapshot(mock, "2026-08-20", 100000, 0, 1, 500000, "N")
	mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(donationConfigRows("N", 0))
	before, err := donationService.GetSummary()
	if err != nil {
		t.Fatalf("initial GetSummary() error = %v", err)
	}
	if before.DisplayAmount != 100000 {
		t.Fatalf("initial display amount = %d, want 100000", before.DisplayAmount)
	}

	mock.ExpectBegin()
	mock.ExpectExec(`(?s)INSERT INTO WEO_ORDER \(.*O_TYPE.*\) VALUES`).
		WillReturnResult(sqlmock.NewResult(3001, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`(?s)FROM WEO_ORDER o.*WHERE o.O_SEQ = \? AND o.O_TYPE = 'A'`).
		WithArgs(int64(3001)).
		WillReturnRows(donationOrderRows().AddRow(
			int64(3001), nil, "other", nil, "2026-08-20",
			"추가 기부자", "20", "영어과", "01012345678", "one_time",
			int64(50000), int64(0), int64(50000), "completed", "admin", nil,
			7, "2026-08-20T07:10:00Z", "192.0.2.1",
		))

	order, err := orchestrator.CreateOrder(validSummaryDonationOrderInput(), 7, "192.0.2.1")
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if order.OrderSeq != 3001 || snapshotCreator.calls != 1 {
		t.Fatalf("created order = %+v, snapshot refresh calls = %d", order, snapshotCreator.calls)
	}

	expectSnapshot(mock, "2026-08-20", 150000, 0, 2, 500000, "N")
	mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(donationConfigRows("N", 0))
	after, err := donationService.GetSummary()
	if err != nil {
		t.Fatalf("refreshed GetSummary() error = %v", err)
	}
	if after.DisplayAmount != 150000 || after.DonorCount != 2 {
		t.Fatalf("refreshed summary = %+v", after)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotRefreshFailureForcesCanonicalLiveSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	donationRepo := repository.NewDonationRepository(sqlxDB)
	donationService := service.NewDonationService(donationRepo, cache.New(5*time.Minute, 10*time.Minute))
	adminService := service.NewAdminDonationService(repository.NewAdminDonationRepository(sqlxDB), donationRepo)
	orchestrator := service.NewDonationConfigOrchestrator(
		adminService,
		donationService,
		&snapshotCreatorStub{err: errors.New("snapshot unavailable")},
	)

	mock.ExpectBegin()
	mock.ExpectExec(`(?s)INSERT INTO WEO_ORDER \(.*O_TYPE.*\) VALUES`).WillReturnResult(sqlmock.NewResult(3001, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`(?s)FROM WEO_ORDER o.*WHERE o.O_SEQ = \? AND o.O_TYPE = 'A'`).
		WillReturnRows(donationOrderRows().AddRow(
			int64(3001), nil, "other", nil, "2026-08-20",
			"추가 기부자", "20", "영어과", "01012345678", "one_time",
			int64(50000), int64(0), int64(50000), "completed", "admin", nil,
			7, "2026-08-20T07:10:00Z", "192.0.2.1",
		))
	if _, err := orchestrator.CreateOrder(validSummaryDonationOrderInput(), 7, "192.0.2.1"); err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*COUNT\(DISTINCT CASE.*O_TYPE = 'A'`).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL_AMOUNT", "DONOR_COUNT"}).AddRow(int64(50000), 1))
	mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(donationConfigRows("N", 0))
	summary, err := donationService.GetSummary()
	if err != nil {
		t.Fatalf("live GetSummary() error = %v", err)
	}
	if summary.DisplayAmount != 70000 || summary.DonorCount != 1 {
		t.Fatalf("live summary = %+v", summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDonationOrderCreateResponseReadFailureStillRefreshesSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	donationRepo := repository.NewDonationRepository(sqlxDB)
	donationService := service.NewDonationService(donationRepo, cache.New(5*time.Minute, 10*time.Minute))
	adminService := service.NewAdminDonationService(repository.NewAdminDonationRepository(sqlxDB), donationRepo)
	snapshotCreator := &snapshotCreatorStub{}
	orchestrator := service.NewDonationConfigOrchestrator(adminService, donationService, snapshotCreator)
	responseReadErr := errors.New("response read failed")

	mock.ExpectBegin()
	mock.ExpectExec(`(?s)INSERT INTO WEO_ORDER \(.*O_TYPE.*\) VALUES`).WillReturnResult(sqlmock.NewResult(3001, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`(?s)FROM WEO_ORDER o.*WHERE o.O_SEQ = \? AND o.O_TYPE = 'A'`).
		WithArgs(int64(3001)).
		WillReturnError(responseReadErr)

	_, err = orchestrator.CreateOrder(validSummaryDonationOrderInput(), 7, "192.0.2.1")
	if !errors.Is(err, responseReadErr) {
		t.Fatalf("CreateOrder() error = %v, want response read failure", err)
	}
	if snapshotCreator.calls != 1 {
		t.Fatalf("snapshot refresh calls = %d, want 1", snapshotCreator.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDonationOrderUpdateResponseReadFailureStillRefreshesSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	donationRepo := repository.NewDonationRepository(sqlxDB)
	donationService := service.NewDonationService(donationRepo, cache.New(5*time.Minute, 10*time.Minute))
	adminService := service.NewAdminDonationService(repository.NewAdminDonationRepository(sqlxDB), donationRepo)
	snapshotCreator := &snapshotCreatorStub{}
	orchestrator := service.NewDonationConfigOrchestrator(adminService, donationService, snapshotCreator)
	responseReadErr := errors.New("response read failed")
	input := validSummaryDonationOrderInput()
	input.LastEditedAt = "2026-08-20T12:00:00Z"

	mock.ExpectBegin()
	mock.ExpectExec(`(?s)UPDATE WEO_ORDER SET.*WHERE O_SEQ = \? AND O_TYPE = 'A'`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`(?s)FROM WEO_ORDER o.*WHERE o.O_SEQ = \? AND o.O_TYPE = 'A'`).
		WithArgs(int64(3001)).
		WillReturnError(responseReadErr)

	_, err = orchestrator.UpdateOrder(3001, input, 7, "192.0.2.1")
	if !errors.Is(err, responseReadErr) {
		t.Fatalf("UpdateOrder() error = %v, want response read failure", err)
	}
	if snapshotCreator.calls != 1 {
		t.Fatalf("snapshot refresh calls = %d, want 1", snapshotCreator.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDonationConfigUpdateRollsBackWhenSnapshotUpsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	donationRepo := repository.NewDonationRepository(sqlxDB)
	adminService := service.NewAdminDonationService(repository.NewAdminDonationRepository(sqlxDB), donationRepo)
	donationService := service.NewDonationService(donationRepo, cache.New(5*time.Minute, 10*time.Minute))
	snapshotJob := job.NewDonationSnapshotJob(donationRepo, zerolog.Nop())
	orchestrator := service.NewDonationConfigOrchestrator(adminService, donationService, snapshotJob)
	update := service.DonationConfigUpdate{
		Goal: 500000, ManualAdj: 20000, ManualDonorCnt: 4,
		TierSproutMin: 1, TierSaplingMin: 10000, TierTreeMin: 50000,
		TierBloomingMin: 100000, TierFruitingMin: 300000,
		Note: "atomic update",
	}

	mock.ExpectBegin()
	mock.ExpectExec(`(?s)UPDATE DONATION_CONFIG.*WHERE IS_ACTIVE = 'Y'`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*COUNT\(DISTINCT CASE.*O_TYPE = 'A'`).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL_AMOUNT", "DONOR_COUNT"}).AddRow(int64(180000), 3))
	mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(donationConfigRows("N", 0))
	snapshotErr := errors.New("snapshot write failed")
	mock.ExpectExec(`(?s)INSERT INTO DONATION_SNAPSHOT.*ON DUPLICATE KEY UPDATE`).
		WillReturnError(snapshotErr)
	mock.ExpectRollback()

	err = orchestrator.UpdateConfig(update, 7)
	if !errors.Is(err, snapshotErr) {
		t.Fatalf("UpdateConfig() error = %v, want snapshot failure", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func validSummaryDonationOrderInput() model.DonationOrderInput {
	return model.DonationOrderInput{
		Source: "other", AccountUsrSeqSet: true, DonationDate: "2026-08-20",
		Donor:        model.DonationDonor{Name: "추가 기부자", Cohort: "20", Department: "영어과", Phone: "01012345678"},
		DonationType: "one_time", GrossAmount: 50000, Status: "completed", PaymentMethod: "admin",
	}
}

func donationOrderRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"ORDER_SEQ", "ACCOUNT_USR_SEQ", "SOURCE", "TRANSACTION_NUMBER", "DONATION_DATE",
		"DONOR_NAME", "DONOR_COHORT", "DONOR_DEPARTMENT", "DONOR_PHONE", "DONATION_TYPE",
		"GROSS_AMOUNT", "REFUNDED_AMOUNT", "NET_RECEIVED_AMOUNT", "STATUS", "PAYMENT_METHOD", "MEMO",
		"LAST_EDITED_BY", "LAST_EDITED_AT", "LAST_EDITED_IP",
	})
}
