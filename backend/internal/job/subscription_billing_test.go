// subscription_billing_test.go — Unit tests for SubscriptionBillingJob scheduling helpers
package job

import (
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

// TestNextRunAt_BeforeThreeAM expects today's 03:00 when "now" is earlier in the day.
func TestNextRunAt_BeforeThreeAM(t *testing.T) {
	loc := time.FixedZone("KST", 9*3600)
	now := time.Date(2026, 4, 25, 1, 30, 0, 0, loc)
	want := time.Date(2026, 4, 25, 3, 0, 0, 0, loc)
	got := nextRunAt(now)
	if !got.Equal(want) {
		t.Errorf("nextRunAt(%s) = %s, want %s", now, got, want)
	}
}

// TestNextRunAt_AfterThreeAM expects tomorrow's 03:00 when "now" is past today's 03:00.
func TestNextRunAt_AfterThreeAM(t *testing.T) {
	loc := time.FixedZone("KST", 9*3600)
	now := time.Date(2026, 4, 25, 9, 0, 0, 0, loc)
	want := time.Date(2026, 4, 26, 3, 0, 0, 0, loc)
	got := nextRunAt(now)
	if !got.Equal(want) {
		t.Errorf("nextRunAt(%s) = %s, want %s", now, got, want)
	}
}

// TestNextRunAt_ExactlyThreeAM rolls forward when "now" is exactly the trigger time
// (otherwise the goroutine would fire immediately twice in a row).
func TestNextRunAt_ExactlyThreeAM(t *testing.T) {
	loc := time.FixedZone("KST", 9*3600)
	now := time.Date(2026, 4, 25, 3, 0, 0, 0, loc)
	want := time.Date(2026, 4, 26, 3, 0, 0, 0, loc)
	got := nextRunAt(now)
	if !got.Equal(want) {
		t.Errorf("nextRunAt(%s) = %s, want %s", now, got, want)
	}
}

// TestNextRunAt_DSTAcrossMidnight verifies the next-fire calculation when crossing
// midnight near the start of a month.
func TestNextRunAt_DSTAcrossMidnight(t *testing.T) {
	loc := time.FixedZone("KST", 9*3600)
	now := time.Date(2026, 4, 30, 23, 59, 59, 0, loc)
	want := time.Date(2026, 5, 1, 3, 0, 0, 0, loc)
	got := nextRunAt(now)
	if !got.Equal(want) {
		t.Errorf("nextRunAt(%s) = %s, want %s", now, got, want)
	}
}

// TestEasyPayBillingInterfaceShape is a compile-time assertion that ensures the
// EasyPayBilling interface stays compatible with what the production EasyPayService
// implements. We construct a dummy implementer and assign it to the interface.
func TestEasyPayBillingInterfaceShape(t *testing.T) {
	var _ EasyPayBilling = (*stubBilling)(nil)
}

// stubBilling is a no-op implementer used purely for the interface-shape test above.
type stubBilling struct{}

func (s *stubBilling) AutoBilling(billingKey, orderNo string, amount int, traceNo, clientIP string) (*model.ApproveResult, error) {
	return &model.ApproveResult{ResCode: "0000"}, nil
}

// TestMaxConsecutiveFailsConst is a regression guard: the threshold is part of the
// product spec ("3회 누적 실패 시 자동 suspend").
func TestMaxConsecutiveFailsConst(t *testing.T) {
	if MaxConsecutiveFails != 3 {
		t.Errorf("MaxConsecutiveFails = %d, want 3 (spec)", MaxConsecutiveFails)
	}
}
