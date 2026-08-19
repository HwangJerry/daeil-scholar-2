package model

// PersonalDonationItem represents a received donation linked to a member account.
type PersonalDonationItem struct {
	OrderSeq          int64  `db:"ORDER_SEQ" json:"orderSeq"`
	DonationDate      string `db:"DONATION_DATE" json:"donationDate"`
	GrossAmount       int64  `db:"GROSS_AMOUNT" json:"grossAmount"`
	RefundedAmount    int64  `db:"REFUNDED_AMOUNT" json:"refundedAmount"`
	NetReceivedAmount int64  `db:"NET_RECEIVED_AMOUNT" json:"netReceivedAmount"`
	LifecycleStatus   string `db:"LIFECYCLE_STATUS" json:"lifecycleStatus"`
	PaymentMethod     string `db:"PAYMENT_METHOD" json:"paymentMethod"`
	Source            string `db:"SOURCE" json:"source"`
}

// PersonalDonationSummary is the paginated response for GET /api/donation/my.
// TotalAmount retains the legacy response name while representing canonical net receipts.
type PersonalDonationSummary struct {
	Items       []PersonalDonationItem `json:"items"`
	TotalAmount int64                  `json:"totalAmount"`
	TotalCount  int                    `json:"totalCount"`
	Page        int                    `json:"page"`
	Size        int                    `json:"size"`
}
