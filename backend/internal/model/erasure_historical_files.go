// erasure_historical_files.go — Reviewed ownership evidence for orphaned uploads.
package model

// Operational input only; never expose these paths through public receipts.
type ErasureHistoricalFilePlan struct {
	RequestID          int64    `json:"requestId"`
	UserSeq            int      `json:"userSeq"`
	Evidence           string   `json:"evidenceReference"`
	OwnershipVerified  bool     `json:"ownershipVerified"`
	RetentionRespected bool     `json:"retentionRespected"`
	Files              []string `json:"files"`
}
