package service

import (
	"testing"

	"github.com/dflh-saf/backend/internal/model"
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
