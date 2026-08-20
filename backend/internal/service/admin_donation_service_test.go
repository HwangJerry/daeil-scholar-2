package service

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestNormalizeDonationOrderInputPartiallyRefunded(t *testing.T) {
	input := model.DonationOrderInput{
		Source:         "bank_transfer",
		DonationDate:   "2026-07-28",
		Donor:          model.DonationDonor{Name: "예시 동문", Cohort: "18", Department: "영어", Phone: "010-1234-5678"},
		DonationType:   "one_time",
		GrossAmount:    100000,
		RefundedAmount: 20000,
		Status:         "partially_refunded",
		PaymentMethod:  "bank",
	}

	normalized, err := NormalizeDonationOrderInput(input)
	if err != nil {
		t.Fatalf("NormalizeDonationOrderInput() error = %v", err)
	}
	if normalized.NetReceivedAmount != 80000 {
		t.Fatalf("net = %d, want 80000", normalized.NetReceivedAmount)
	}
	if normalized.Donor.Phone != "01012345678" {
		t.Fatalf("phone = %q, want normalized phone", normalized.Donor.Phone)
	}
	if normalized.LegacyGate != "S" || normalized.LegacyStatus != "Y" || normalized.LegacyPayment != "Y" {
		t.Fatalf("legacy mapping = %q/%q/%q, want S/Y/Y", normalized.LegacyGate, normalized.LegacyStatus, normalized.LegacyPayment)
	}
	if normalized.CompositeKey == "" {
		t.Fatal("composite key must be generated when transactionNumber is absent")
	}
}

func TestNormalizeDonationOrderInputRejectsInconsistentAmounts(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		gross    int64
		refunded int64
	}{
		{name: "negative gross", status: "pending", gross: -1, refunded: 0},
		{name: "refund exceeds gross", status: "partially_refunded", gross: 100, refunded: 101},
		{name: "completed with refund", status: "completed", gross: 100, refunded: 1},
		{name: "partial without refund", status: "partially_refunded", gross: 100, refunded: 0},
		{name: "partial equals gross", status: "partially_refunded", gross: 100, refunded: 100},
		{name: "full refund differs from gross", status: "fully_refunded", gross: 100, refunded: 99},
		{name: "pending with refund", status: "pending", gross: 100, refunded: 1},
		{name: "cancelled with refund", status: "cancelled", gross: 100, refunded: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeDonationOrderInput(model.DonationOrderInput{
				Source:         "other",
				DonationDate:   "2026-07-28",
				Donor:          model.DonationDonor{Name: "기부자", Cohort: "18", Department: "영어", Phone: "01000000000"},
				DonationType:   "one_time",
				GrossAmount:    tt.gross,
				RefundedAmount: tt.refunded,
				Status:         tt.status,
				PaymentMethod:  "admin",
			})
			if err == nil {
				t.Fatal("NormalizeDonationOrderInput() error = nil, want validation error")
			}
		})
	}
}

func TestNormalizeDonationOrderInputRejectsInvalidCanonicalFields(t *testing.T) {
	transaction := "TX-1"
	base := model.DonationOrderInput{
		Source:            "bank_transfer",
		TransactionNumber: &transaction,
		DonationDate:      "2026-07-28",
		Donor:             model.DonationDonor{Name: "기부자", Cohort: "18", Department: "영어", Phone: "01000000000"},
		DonationType:      "one_time",
		GrossAmount:       100,
		Status:            "completed",
		PaymentMethod:     "bank",
	}
	tests := []struct {
		name   string
		mutate func(*model.DonationOrderInput)
	}{
		{name: "unknown source", mutate: func(v *model.DonationOrderInput) { v.Source = "legacy" }},
		{name: "invalid date", mutate: func(v *model.DonationOrderInput) { v.DonationDate = "2026-02-30" }},
		{name: "missing donor name", mutate: func(v *model.DonationOrderInput) { v.Donor.Name = " " }},
		{name: "missing cohort", mutate: func(v *model.DonationOrderInput) { v.Donor.Cohort = "" }},
		{name: "missing department", mutate: func(v *model.DonationOrderInput) { v.Donor.Department = "" }},
		{name: "invalid phone", mutate: func(v *model.DonationOrderInput) { v.Donor.Phone = "02-123-4567" }},
		{name: "unknown donation type", mutate: func(v *model.DonationOrderInput) { v.DonationType = "once" }},
		{name: "unknown status", mutate: func(v *model.DonationOrderInput) { v.Status = "paid" }},
		{name: "unknown payment method", mutate: func(v *model.DonationOrderInput) { v.PaymentMethod = "cash" }},
		{name: "blank transaction number", mutate: func(v *model.DonationOrderInput) { blank := " "; v.TransactionNumber = &blank }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := base
			tc.mutate(&input)
			if _, err := NormalizeDonationOrderInput(input); err == nil {
				t.Fatal("NormalizeDonationOrderInput() error = nil, want validation error")
			}
		})
	}
}

func TestListOrdersRejectsUnknownCanonicalFilter(t *testing.T) {
	service := NewAdminDonationService(nil, nil)
	_, err := service.ListOrders(model.DonationOrderFilters{Source: "legacy"}, 1, 20)
	if err == nil {
		t.Fatal("ListOrders() error = nil, want validation error")
	}
}

func TestUpdateConfigRejectsInvalidDonationTierThresholds(t *testing.T) {
	tests := []struct {
		name   string
		update DonationConfigUpdate
	}{
		{
			name: "negative",
			update: DonationConfigUpdate{
				TierSproutMin: -1, TierSaplingMin: 10000, TierTreeMin: 50000,
				TierBloomingMin: 100000, TierFruitingMin: 300000,
			},
		},
		{
			name: "equal",
			update: DonationConfigUpdate{
				TierSproutMin: 1, TierSaplingMin: 10000, TierTreeMin: 50000,
				TierBloomingMin: 50000, TierFruitingMin: 300000,
			},
		},
		{
			name: "descending",
			update: DonationConfigUpdate{
				TierSproutMin: 1, TierSaplingMin: 50000, TierTreeMin: 10000,
				TierBloomingMin: 100000, TierFruitingMin: 300000,
			},
		},
	}

	service := NewAdminDonationService(nil, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UpdateConfig(tt.update, 7)
			if !errors.Is(err, ErrInvalidTierThresholds) {
				t.Fatalf("UpdateConfig() error = %v, want ErrInvalidTierThresholds", err)
			}
		})
	}
}

func TestUpdateConfigPersistsValidDonationTierThresholds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewAdminDonationService(repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock")), nil)
	update := DonationConfigUpdate{
		Goal: 250000000, ManualAdj: 5000, ManualDonorCnt: 12,
		TierSproutMin: 10, TierSaplingMin: 20000, TierTreeMin: 60000,
		TierBloomingMin: 120000, TierFruitingMin: 350000,
		Note: "tier update", Overwrite: true,
	}

	mock.ExpectExec(`UPDATE DONATION_CONFIG`).
		WithArgs(
			update.Goal, update.ManualAdj, update.ManualDonorCnt,
			update.TierSproutMin, update.TierSaplingMin, update.TierTreeMin,
			update.TierBloomingMin, update.TierFruitingMin,
			update.Note, "Y", 7,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := service.UpdateConfig(update, 7); err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateOrderRejectsInvalidDonationAccountSequence(t *testing.T) {
	accountUsrSeq := 0
	service := NewAdminDonationService(nil, nil)

	_, err := service.CreateOrder(model.DonationOrderInput{
		Source:         "other",
		AccountUsrSeq:  &accountUsrSeq,
		DonationDate:   "2026-07-28",
		Donor:          model.DonationDonor{Name: "기부자", Cohort: "18", Department: "영어", Phone: "01000000000"},
		DonationType:   "one_time",
		GrossAmount:    100000,
		RefundedAmount: 0,
		Status:         "completed",
		PaymentMethod:  "admin",
	}, 7, "192.0.2.1")

	if !errors.Is(err, ErrInvalidDonationOrder) {
		t.Fatalf("CreateOrder() error = %v, want ErrInvalidDonationOrder", err)
	}
}

func TestCreateOrderRejectsMissingDonationAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewAdminDonationService(repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock")), nil)
	accountUsrSeq := 9999

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*USR_STATUS != 'AAA'[\s\S]*FOR UPDATE`).
		WithArgs(accountUsrSeq).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}))
	mock.ExpectRollback()

	_, err = service.CreateOrder(validDonationOrderInput(&accountUsrSeq, true), 7, "192.0.2.1")
	if !errors.Is(err, ErrDonationAccountNotFound) {
		t.Fatalf("CreateOrder() error = %v, want ErrDonationAccountNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateOrderRejectsInvalidDonationAccountSequence(t *testing.T) {
	accountUsrSeq := 0
	service := NewAdminDonationService(nil, nil)

	err := service.UpdateOrder(3001, model.DonationOrderInput{
		Source:           "other",
		AccountUsrSeq:    &accountUsrSeq,
		AccountUsrSeqSet: true,
		DonationDate:     "2026-07-28",
		Donor:            model.DonationDonor{Name: "기부자", Cohort: "18", Department: "영어", Phone: "01000000000"},
		DonationType:     "one_time",
		GrossAmount:      100000,
		RefundedAmount:   0,
		Status:           "completed",
		PaymentMethod:    "admin",
	}, 7, "192.0.2.1")

	if !errors.Is(err, ErrInvalidDonationOrder) {
		t.Fatalf("UpdateOrder() error = %v, want ErrInvalidDonationOrder", err)
	}
}

func TestUpdateOrderRejectsMissingDonationAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewAdminDonationService(repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock")), nil)
	accountUsrSeq := 9999

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*USR_STATUS != 'AAA'[\s\S]*FOR UPDATE`).
		WithArgs(accountUsrSeq).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}))
	mock.ExpectRollback()

	err = service.UpdateOrder(3001, validDonationOrderInput(&accountUsrSeq, true), 7, "192.0.2.1")
	if !errors.Is(err, ErrDonationAccountNotFound) {
		t.Fatalf("UpdateOrder() error = %v, want ErrDonationAccountNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func validDonationOrderInput(accountUsrSeq *int, accountUsrSeqSet bool) model.DonationOrderInput {
	return model.DonationOrderInput{
		Source:           "other",
		AccountUsrSeq:    accountUsrSeq,
		AccountUsrSeqSet: accountUsrSeqSet,
		DonationDate:     "2026-07-28",
		Donor:            model.DonationDonor{Name: "기부자", Cohort: "18", Department: "영어", Phone: "01000000000"},
		DonationType:     "one_time",
		GrossAmount:      100000,
		Status:           "completed",
		PaymentMethod:    "admin",
		LastEditedAt:     "2026-08-20T12:00:00Z",
	}
}
