// account_erasure_preview.go — Read-only erasure plan an operator reviews before approving.
package model

import "time"

const (
	ErasureActionDelete    = "delete"
	ErasureActionAnonymize = "anonymize"
)

// ErasurePreview lists the application records the automatic erasure would
// change for one request and how each is changed. PlanDigest binds an operator
// approval to exactly this set of records; any later change requires re-review.
type ErasurePreview struct {
	RequestID   int64                      `json:"requestId"`
	GeneratedAt time.Time                  `json:"generatedAt"`
	PlanDigest  string                     `json:"planDigest"`
	Blockers    []string                   `json:"blockers"`
	Tables      []ErasurePreviewTable      `json:"tables"`
	Files       []string                   `json:"files"`
	Unhandled   []AccountDeletionFootprint `json:"unhandled"`
}

// ErasurePreviewTable is one table's affected rows. Rows is a bounded sample;
// Count is the full number of rows the erasure would change.
type ErasurePreviewTable struct {
	Table          string              `json:"table"`
	Action         string              `json:"action"`
	Columns        []string            `json:"columns"`
	MaskedColumns  []string            `json:"maskedColumns"`
	ChangedColumns []string            `json:"changedColumns"`
	Count          int64               `json:"count"`
	Rows           []ErasurePreviewRow `json:"rows"`
}

// ErasurePreviewRow aligns values with ErasurePreviewTable.Columns. After is set
// only for anonymized rows. A nil value is SQL NULL.
type ErasurePreviewRow struct {
	Before []*string `json:"before"`
	After  []*string `json:"after,omitempty"`
	Note   string    `json:"note,omitempty"`
}
