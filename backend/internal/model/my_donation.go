// my_donation.go — model types for user donation history
package model

// MyDonationItem represents a single paid donation order for the user.
type MyDonationItem struct {
	OrderSeq int    `json:"orderSeq" db:"OrderSeq"`
	Amount   int    `json:"amount" db:"Amount"`
	PayType  string `json:"payType" db:"PayType"`
	PaidAt   string `json:"paidAt" db:"PaidAt"`
}

// MyDonationResponse is the API response for GET /api/donation/my.
type MyDonationResponse struct {
	Items       []MyDonationItem `json:"items"`
	TotalAmount int64            `json:"totalAmount"`
	TotalCount  int              `json:"totalCount"`
}
