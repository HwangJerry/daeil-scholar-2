package model

// DonationSummary is the API response for GET /api/donation/summary.
type DonationSummary struct {
	TotalAmount     int64   `json:"totalAmount"`
	ManualAdj       int64   `json:"manualAdj"`
	DisplayAmount   int64   `json:"displayAmount"`
	DonorCount      int     `json:"donorCount"`
	GoalAmount      int64   `json:"goalAmount"`
	AchievementRate float64 `json:"achievementRate"`
	SnapshotDate    string  `json:"snapshotDate"`
}
