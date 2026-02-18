// subscription.go — Domain model for recurring donation subscriptions
package model

import "time"

// Subscription represents a recurring donation subscription record.
type Subscription struct {
	SubSeq    int       `json:"subSeq" db:"SUB_SEQ"`
	USRSeq    int       `json:"usrSeq" db:"USR_SEQ"`
	Amount    int       `json:"amount" db:"AMOUNT"`
	PayType   string    `json:"payType" db:"PAY_TYPE"`
	Status    string    `json:"status" db:"STATUS"`
	StartDate time.Time `json:"startDate" db:"START_DATE"`
	NextBill  time.Time `json:"nextBill" db:"NEXT_BILL"`
	RegDate   time.Time `json:"regDate" db:"REG_DATE"`
}

// CreateSubscriptionRequest is the request body for POST /api/donation/subscription.
type CreateSubscriptionRequest struct {
	Amount  int    `json:"amount"`
	PayType string `json:"payType"`
}
