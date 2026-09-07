// account_erasure_target.go — Storage-specific progress, without user identifiers.
package model

import "time"

var ErasureTargetNames = []string{"backups", "historical_files", "external_data", "other_identifiers"}

type ErasureTarget struct {
	Name          string     `json:"target" db:"TARGET"`
	Status        string     `json:"status" db:"STATUS"`
	Evidence      string     `json:"evidenceReference" db:"EVIDENCE_REFERENCE"`
	Code          string     `json:"code" db:"LAST_CODE"`
	Attempts      int        `json:"attempts" db:"ATTEMPTS"`
	LastAttemptAt *time.Time `json:"lastAttemptAt" db:"LAST_ATTEMPT_AT"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"UPDATED_AT"`
}

func (t ErasureTarget) Verified() bool {
	return (t.Status == "complete" || t.Status == "not_applicable") && t.Evidence != ""
}
