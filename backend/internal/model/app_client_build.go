// app_client_build.go — Build numbers observed from mobile API requests.
package model

import "time"

// AppClientBuild is one platform/build pair seen by the API. It carries no user
// or device identifier: it exists only so administrators can pick a real build
// number when setting update thresholds.
type AppClientBuild struct {
	Platform    string    `db:"PLATFORM" json:"platform"`
	Build       int64     `db:"BUILD" json:"build"`
	VersionName string    `db:"VERSION_NAME" json:"versionName"`
	FirstSeenAt time.Time `db:"FIRST_SEEN_AT" json:"firstSeenAt"`
	LastSeenAt  time.Time `db:"LAST_SEEN_AT" json:"lastSeenAt"`
}
