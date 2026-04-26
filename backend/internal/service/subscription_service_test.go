// subscription_service_test.go — Unit tests for subscription helpers and validation
package service

import (
	"testing"
	"time"
)

// TestComputeNextBillDate verifies the next-bill helper used at activation and at every
// monthly auto-billing tick.
func TestComputeNextBillDate(t *testing.T) {
	loc := time.FixedZone("KST", 9*3600)
	cases := []struct {
		name    string
		from    time.Time
		billDay int
		wantYMD string
	}{
		{
			name:    "normal mid-month carries day forward",
			from:    time.Date(2026, 4, 15, 12, 0, 0, 0, loc),
			billDay: 15,
			wantYMD: "2026-05-15",
		},
		{
			name:    "billDay zero clamps to 1",
			from:    time.Date(2026, 4, 15, 12, 0, 0, 0, loc),
			billDay: 0,
			wantYMD: "2026-05-01",
		},
		{
			name:    "billDay over 28 clamps to 28",
			from:    time.Date(2026, 4, 15, 12, 0, 0, 0, loc),
			billDay: 31,
			wantYMD: "2026-05-28",
		},
		{
			name:    "crosses year boundary",
			from:    time.Date(2026, 12, 10, 12, 0, 0, 0, loc),
			billDay: 10,
			wantYMD: "2027-01-10",
		},
		{
			name:    "from late in month with billDay=1",
			from:    time.Date(2026, 4, 30, 23, 59, 59, 0, loc),
			billDay: 1,
			wantYMD: "2026-06-01",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeNextBillDate(tc.from, tc.billDay).Format("2006-01-02")
			if got != tc.wantYMD {
				t.Errorf("ComputeNextBillDate(%s, %d) = %s, want %s",
					tc.from.Format("2006-01-02"), tc.billDay, got, tc.wantYMD)
			}
		})
	}
}

// TestEasyPayServiceImplementsCanceler is a compile-time assertion that the production
// EasyPayService still satisfies the EasyPayCanceler interface relied on by
// SubscriptionService.CancelSubscription. If somebody renames RevokeBillingKey, this
// test fails at build time.
func TestEasyPayServiceImplementsCanceler(t *testing.T) {
	var _ EasyPayCanceler = (*EasyPayService)(nil)
}
