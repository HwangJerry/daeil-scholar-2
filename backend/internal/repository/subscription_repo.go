// subscription_repo.go — Database access for subscription records
package repository

import (
	"database/sql"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// SubscriptionRepository handles CRUD operations for the SUBSCRIPTION table.
type SubscriptionRepository struct {
	DB *sqlx.DB
}

// NewSubscriptionRepository creates a new SubscriptionRepository.
func NewSubscriptionRepository(db *sqlx.DB) *SubscriptionRepository {
	return &SubscriptionRepository{DB: db}
}

// Insert creates a new subscription record and returns the auto-generated ID.
func (r *SubscriptionRepository) Insert(sub *model.Subscription) (int64, error) {
	result, err := r.DB.Exec(`
		INSERT INTO SUBSCRIPTION (USR_SEQ, AMOUNT, PAY_TYPE, STATUS, START_DATE, NEXT_BILL, REG_DATE)
		VALUES (?, ?, ?, ?, ?, ?, NOW())
	`, sub.USRSeq, sub.Amount, sub.PayType, sub.Status, sub.StartDate, sub.NextBill)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetByUser returns the active subscription for a given user, or nil if none exists.
func (r *SubscriptionRepository) GetByUser(usrSeq int) (*model.Subscription, error) {
	var sub model.Subscription
	err := r.DB.Get(&sub, `
		SELECT SUB_SEQ, USR_SEQ, AMOUNT, PAY_TYPE, STATUS, START_DATE, NEXT_BILL, REG_DATE
		FROM SUBSCRIPTION
		WHERE USR_SEQ = ? AND STATUS = 'active'
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

// UpdateStatus sets a subscription's status and records the edit timestamp.
func (r *SubscriptionRepository) UpdateStatus(subSeq int, status string) error {
	_, err := r.DB.Exec(`
		UPDATE SUBSCRIPTION SET STATUS = ?, EDT_DATE = NOW() WHERE SUB_SEQ = ?
	`, status, subSeq)
	return err
}
