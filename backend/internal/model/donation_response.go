package model

// DonationSummary is the API response for GET /api/donation/summary.
type DonationSummary struct {
	DisplayAmount   int64                  `json:"displayAmount"`
	GoalAmount      int64                  `json:"goalAmount"`
	DonorCount      int                    `json:"donorCount"`
	AchievementRate float64                `json:"achievementRate"`
	SnapshotDate    string                 `json:"snapshotDate"`
	TierThresholds  DonationTierThresholds `json:"tierThresholds"`
}

type DonationTierThresholds struct {
	Sprout   int64 `json:"sprout"`
	Sapling  int64 `json:"sapling"`
	Tree     int64 `json:"tree"`
	Blooming int64 `json:"blooming"`
	Fruiting int64 `json:"fruiting"`
}
