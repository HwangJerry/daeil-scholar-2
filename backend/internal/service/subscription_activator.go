// subscription_activator.go — Activates a pending recurring subscription after its first PG approval
package service

import (
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

// SubscriptionActivator owns the post-first-payment lifecycle of a recurring subscription
// (lookup, fail-mark, activate) so the donation/PG flow does not have to know how billing
// keys, fail-counts, or next-bill dates are persisted.
type SubscriptionActivator struct {
	subRepo *repository.SubscriptionRepository
}

// NewSubscriptionActivator builds a SubscriptionActivator from a subscription repository.
func NewSubscriptionActivator(subRepo *repository.SubscriptionRepository) *SubscriptionActivator {
	return &SubscriptionActivator{subRepo: subRepo}
}

// LookupPendingByOrder returns the subscription whose first-payment order matches `orderSeq`.
// nil indicates no pending subscription is tied to this order (e.g. a one-time donation).
func (a *SubscriptionActivator) LookupPendingByOrder(orderSeq int) *model.Subscription {
	sub, _ := a.subRepo.GetByOrderSeq(orderSeq)
	return sub
}

// MarkFirstPaymentFailed transitions a pending subscription to 'failed'. Errors are
// swallowed because a failed status update must not mask the upstream PG failure
// the caller is already reporting.
func (a *SubscriptionActivator) MarkFirstPaymentFailed(subSeq int) {
	_ = a.subRepo.MarkFailed(subSeq)
}

// ActivateAfterFirstPayment persists the issued billing key, masked card, and next bill
// date for `sub` within the caller-provided tx. Caller is expected to own the surrounding
// transaction and to roll back on error.
func (a *SubscriptionActivator) ActivateAfterFirstPayment(
	tx *sqlx.Tx,
	sub *model.Subscription,
	result *model.ApproveResult,
	now time.Time,
) error {
	billDay := 1
	if sub.BillDay.Valid {
		billDay = int(sub.BillDay.Int64)
	}
	nextBill := ComputeNextBillDate(now, billDay)
	return a.subRepo.ActivateWithBillingKeyTx(tx, sub.SubSeq, result.CNO, result.CardNo, nextBill)
}

// ComputeNextBillDate returns the next billing datetime one month after `from`, pinned to
// `billDay`. billDay is clamped to 1..28 so February is always reachable. Exported because
// the daily auto-billing batch job uses the same calculation.
func ComputeNextBillDate(from time.Time, billDay int) time.Time {
	if billDay < 1 {
		billDay = 1
	}
	if billDay > 28 {
		billDay = 28
	}
	next := from.AddDate(0, 1, 0)
	return time.Date(next.Year(), next.Month(), billDay, 0, 0, 0, 0, next.Location())
}
