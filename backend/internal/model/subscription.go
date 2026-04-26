// subscription.go — Domain model for recurring donation subscriptions
package model

import (
	"database/sql"
	"time"
)

// Subscription represents a recurring donation subscription record.
// BillingKey is intentionally json:"-" — never expose stored billing keys to clients.
type Subscription struct {
	SubSeq       int            `json:"subSeq"       db:"SUB_SEQ"`
	USRSeq       int            `json:"usrSeq"       db:"USR_SEQ"`
	Amount       int            `json:"amount"       db:"AMOUNT"`
	PayType      string         `json:"payType"      db:"PAY_TYPE"`
	BillingKey   sql.NullString `json:"-"            db:"BILLING_KEY"`
	CardNo       sql.NullString `json:"cardNo"       db:"CARD_NO"`
	Status       string         `json:"status"       db:"STATUS"`
	StartDate    time.Time      `json:"startDate"    db:"START_DATE"`
	NextBill     time.Time      `json:"nextBill"     db:"NEXT_BILL"`
	BillDay      sql.NullInt64  `json:"billDay"      db:"BILL_DAY"`
	EndYYYYMM    sql.NullString `json:"endYyyymm"    db:"END_YYYYMM"`
	LastBilledAt sql.NullTime   `json:"lastBilledAt" db:"LAST_BILLED_AT"`
	FailCount    int            `json:"failCount"    db:"FAIL_COUNT"`
	OrderSeq     sql.NullInt64  `json:"-"            db:"ORDER_SEQ"`
	RegDate      time.Time      `json:"regDate"      db:"REG_DATE"`
}

// CreateSubscriptionRequest is the request body for POST /api/donation/subscription.
type CreateSubscriptionRequest struct {
	Amount  int    `json:"amount"`
	PayType string `json:"payType"`
}
