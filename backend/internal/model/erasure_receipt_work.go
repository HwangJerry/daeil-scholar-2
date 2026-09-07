// erasure_receipt_work.go — Track receipt work without exposing donor contact information.
package model

import "time"

type ErasureReceiptWork struct {
	Status          string     `db:"STATUS" json:"status"`
	OriginalStorage string     `db:"ORIGINAL_STORAGE" json:"originalStorage"`
	Evidence        string     `db:"EVIDENCE_REFERENCE" json:"evidenceReference"`
	UpdatedAt       time.Time  `db:"UPDATED_AT" json:"updatedAt"`
	CompletedAt     *time.Time `db:"COMPLETED_AT" json:"completedAt"`
}
