package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"testing"
	"time"
)

func TestDeletionReceiptUTCStoredDatesIgnoreLegacyDriverLocation(t *testing.T) {
	seoul, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		t.Fatal(err)
	}
	stored := time.Date(2026, 9, 6, 23, 0, 0, 0, seoul)
	receipt := model.AccountDeletionReceipt{RequestedAt: stored, TargetAt: stored.AddDate(0, 0, 3), DueAt: stored.AddDate(0, 0, 10), CompletedAt: &stored}
	normalizeDeletionReceiptTimes(&receipt)
	if receipt.RequestedAt.Format(time.RFC3339) != "2026-09-06T23:00:00Z" || receipt.CompletedAt.Format(time.RFC3339) != "2026-09-06T23:00:00Z" {
		t.Fatal("legacy driver shifted UTC timestamps")
	}
	if receipt.DueAt.Sub(receipt.RequestedAt) != 10*24*time.Hour {
		t.Fatal("deadline changed")
	}
}
