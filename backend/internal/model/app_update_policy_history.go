// app_update_policy_history.go — Audit entry for one app update policy change.
package model

import "time"

type AppUpdatePolicyHistoryEntry struct {
	Seq        int       `db:"AUPH_SEQ" json:"seq"`
	Platform   string    `db:"PLATFORM" json:"platform"`
	BeforeJSON string    `db:"BEFORE_JSON" json:"before"`
	AfterJSON  string    `db:"AFTER_JSON" json:"after"`
	ChangedBy  *int      `db:"CHANGED_BY" json:"changedBy"`
	ChangedAt  time.Time `db:"CHANGED_AT" json:"changedAt"`
}
