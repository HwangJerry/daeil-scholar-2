// visit.go — Visit tracking domain models for DAU/MAU analytics
package model

import "time"

// VisitSummary represents one row in WEO_VISIT_SUMMARY, a pre-aggregated
// daily snapshot written by the nightly visit aggregation job.
type VisitSummary struct {
	Date       time.Time `db:"VS_DATE"`
	DAUTotal   uint32    `db:"VS_DAU_TOTAL"`
	DAUMember  uint32    `db:"VS_DAU_MEMBER"`
	DAUAnon    uint32    `db:"VS_DAU_ANON"`
	MAUTotal   uint32    `db:"VS_MAU_TOTAL"`
	PageViews  uint32    `db:"VS_PAGEVIEWS"`
	RegDate    time.Time `db:"REG_DATE"`
}

// ActiveUsersPoint is a single datapoint on the admin DAU/MAU chart.
type ActiveUsersPoint struct {
	Date string `json:"date"`
	DAU  uint32 `json:"dau"`
	MAU  uint32 `json:"mau"`
}

// ActiveUsersResponse is the payload for GET /api/admin/stats/active-users.
// Points span the requested [from, to] range, padded with zeros for days that
// have no WEO_VISIT_SUMMARY row yet.
type ActiveUsersResponse struct {
	Points     []ActiveUsersPoint `json:"points"`
	DAUToday   uint32             `json:"dauToday"`
	MAUCurrent uint32             `json:"mauCurrent"`
}
