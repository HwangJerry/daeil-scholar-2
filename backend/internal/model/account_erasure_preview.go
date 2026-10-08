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
	CompletionWaits []ErasureCompletionWait     `json:"completionWaits"`
	Subscriptions   []ErasureSubscriptionReview `json:"subscriptions"`
	RequestID       int64                       `json:"requestId"`
	GeneratedAt     time.Time                   `json:"generatedAt"`
	PlanDigest      string                      `json:"planDigest"`
	Blockers        []string                    `json:"blockers"`
	Tables          []ErasurePreviewTable       `json:"tables"`
	Files           []string                    `json:"files"`
	Social          []ErasureSocialUnlink       `json:"socialUnlinks"`
	Unhandled       []AccountDeletionFootprint  `json:"unhandled"`
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

// ErasureSocialUnlink is one linked Apple or Kakao account the erasure asks
// the provider to unlink before the member's records are deleted.
// Status is pending, delivered, failed or missing_credential.
type ErasureSocialUnlink struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
}

// Completion waits describe policy delays, not permission to delete unknown rows.
type ErasureCompletionWait struct {
	Code       string    `json:"code"`
	Count      int64     `json:"count"`
	ExpectedAt time.Time `json:"expectedAt"`
}
type ErasureSubscriptionReview struct {
	SubscriptionID    int    `json:"subscriptionId"`
	Status            string `json:"status"`
	HasBillingKey     bool   `json:"hasBillingKey"`
	CanReview         bool   `json:"canReview"`
	Reviewed          bool   `json:"reviewed"`
	SourceFingerprint string `json:"sourceFingerprint"`
}

const PhoneVerificationRetention = 24 * time.Hour
const PhoneVerificationCleanupInterval = time.Hour
const ErasureRecheckInterval = 5 * time.Minute
const ErasurePollInterval = time.Minute
const PhoneVerificationWaitCode = "PHONE_VERIFICATION_RETENTION_PENDING"
