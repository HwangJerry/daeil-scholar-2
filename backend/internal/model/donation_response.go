package model

// DonationSummary is the API response for GET /api/donation/summary.
type DonationSummary struct {
	DisplayAmount int64 `json:"displayAmount"`
}
