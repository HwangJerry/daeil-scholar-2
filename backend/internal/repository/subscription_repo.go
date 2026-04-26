// subscription_repo.go — Database access for subscription records
package repository

import (
	"database/sql"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

const subscriptionSelectColumns = `
	SUB_SEQ, USR_SEQ, AMOUNT, PAY_TYPE, BILLING_KEY, CARD_NO, STATUS,
	START_DATE, NEXT_BILL, BILL_DAY, END_YYYYMM, LAST_BILLED_AT, FAIL_COUNT, ORDER_SEQ, REG_DATE
`

// SubscriptionRepository handles CRUD operations for the SUBSCRIPTION table.
type SubscriptionRepository struct {
	DB *sqlx.DB
}

// NewSubscriptionRepository creates a new SubscriptionRepository.
func NewSubscriptionRepository(db *sqlx.DB) *SubscriptionRepository {
	return &SubscriptionRepository{DB: db}
}

// Insert creates a new subscription record and returns the auto-generated ID.
// Used at first-payment setup; STATUS should be 'pending' until ConfirmPayment activates it.
func (r *SubscriptionRepository) Insert(sub *model.Subscription) (int64, error) {
	result, err := r.DB.Exec(`
		INSERT INTO SUBSCRIPTION
			(USR_SEQ, AMOUNT, PAY_TYPE, STATUS, START_DATE, NEXT_BILL, BILL_DAY, ORDER_SEQ, FAIL_COUNT, REG_DATE)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, NOW())
	`, sub.USRSeq, sub.Amount, sub.PayType, sub.Status, sub.StartDate, sub.NextBill, sub.BillDay, sub.OrderSeq)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetByUser returns the active OR pending subscription for a given user, or nil if none exists.
// 'pending' rows must also be returned so a user cannot start a duplicate subscription
// while their first-payment PG window is still open.
func (r *SubscriptionRepository) GetByUser(usrSeq int) (*model.Subscription, error) {
	var sub model.Subscription
	err := r.DB.Get(&sub, `
		SELECT `+subscriptionSelectColumns+`
		FROM SUBSCRIPTION
		WHERE USR_SEQ = ? AND STATUS IN ('active', 'pending')
		ORDER BY SUB_SEQ DESC
		LIMIT 1
	`, usrSeq)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// GetByID returns a subscription by primary key.
func (r *SubscriptionRepository) GetByID(subSeq int) (*model.Subscription, error) {
	var sub model.Subscription
	err := r.DB.Get(&sub, `
		SELECT `+subscriptionSelectColumns+`
		FROM SUBSCRIPTION
		WHERE SUB_SEQ = ?
	`, subSeq)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// GetByOrderSeq returns the subscription whose first-payment ORDER_SEQ matches.
// Used by ConfirmPayment to map a PG return back to the pending subscription.
func (r *SubscriptionRepository) GetByOrderSeq(orderSeq int) (*model.Subscription, error) {
	var sub model.Subscription
	err := r.DB.Get(&sub, `
		SELECT `+subscriptionSelectColumns+`
		FROM SUBSCRIPTION
		WHERE ORDER_SEQ = ?
		LIMIT 1
	`, orderSeq)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// ListDueForBilling returns active subscriptions whose BILL_DAY matches today,
// END_YYYYMM is open (NULL or >= currentYYYYMM), and were not already billed this month.
// MariaDB 10.1 compatible (no CTE/window functions).
func (r *SubscriptionRepository) ListDueForBilling(billDay int, currentYYYYMM string) ([]*model.Subscription, error) {
	rows := []*model.Subscription{}
	err := r.DB.Select(&rows, `
		SELECT `+subscriptionSelectColumns+`
		FROM SUBSCRIPTION
		WHERE STATUS = 'active'
		  AND BILL_DAY = ?
		  AND (END_YYYYMM IS NULL OR END_YYYYMM >= ?)
		  AND (LAST_BILLED_AT IS NULL OR DATE_FORMAT(LAST_BILLED_AT, '%Y%m') < ?)
		ORDER BY SUB_SEQ ASC
	`, billDay, currentYYYYMM, currentYYYYMM)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ActivateWithBillingKeyTx flips a pending subscription to active, stores the issued
// billing key + masked card, sets next bill, and records last_billed_at within an existing tx.
func (r *SubscriptionRepository) ActivateWithBillingKeyTx(
	tx *sqlx.Tx,
	subSeq int,
	billingKey, cardNo string,
	nextBill time.Time,
) error {
	_, err := tx.Exec(`
		UPDATE SUBSCRIPTION
		SET STATUS = 'active',
		    BILLING_KEY = ?,
		    CARD_NO = ?,
		    NEXT_BILL = ?,
		    LAST_BILLED_AT = NOW(),
		    FAIL_COUNT = 0,
		    EDT_DATE = NOW()
		WHERE SUB_SEQ = ?
	`, billingKey, cardNo, nextBill, subSeq)
	return err
}

// MarkBilledTx records a successful auto-billing within an existing tx.
func (r *SubscriptionRepository) MarkBilledTx(
	tx *sqlx.Tx,
	subSeq int,
	billedAt, nextBill time.Time,
) error {
	_, err := tx.Exec(`
		UPDATE SUBSCRIPTION
		SET LAST_BILLED_AT = ?,
		    NEXT_BILL = ?,
		    FAIL_COUNT = 0,
		    EDT_DATE = NOW()
		WHERE SUB_SEQ = ?
	`, billedAt, nextBill, subSeq)
	return err
}

// IncrementFailCount atomically bumps FAIL_COUNT and returns the new value.
func (r *SubscriptionRepository) IncrementFailCount(subSeq int) (int, error) {
	if _, err := r.DB.Exec(`
		UPDATE SUBSCRIPTION
		SET FAIL_COUNT = FAIL_COUNT + 1, EDT_DATE = NOW()
		WHERE SUB_SEQ = ?
	`, subSeq); err != nil {
		return 0, err
	}
	var count int
	if err := r.DB.Get(&count, `SELECT FAIL_COUNT FROM SUBSCRIPTION WHERE SUB_SEQ = ?`, subSeq); err != nil {
		return 0, err
	}
	return count, nil
}

// MarkFailed sets status='failed' (used on first-payment failure).
func (r *SubscriptionRepository) MarkFailed(subSeq int) error {
	return r.UpdateStatus(subSeq, "failed")
}

// MarkSuspended sets status='suspended' (used after consecutive auto-billing failures).
func (r *SubscriptionRepository) MarkSuspended(subSeq int) error {
	return r.UpdateStatus(subSeq, "suspended")
}

// UpdateStatus sets a subscription's status and records the edit timestamp.
func (r *SubscriptionRepository) UpdateStatus(subSeq int, status string) error {
	_, err := r.DB.Exec(`
		UPDATE SUBSCRIPTION SET STATUS = ?, EDT_DATE = NOW() WHERE SUB_SEQ = ?
	`, status, subSeq)
	return err
}
