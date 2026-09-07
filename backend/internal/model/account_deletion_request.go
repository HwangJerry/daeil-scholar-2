// account_deletion_request.go — Manual erasure receipts and operator evidence.
package model

import "time"

type AccountDeletionReceipt struct {
	DatabaseErased  bool       `json:"databaseErased" db:"DATABASE_ERASED"`
	ID              int64      `json:"requestId" db:"REQUEST_ID"`
	Status          string     `json:"status" db:"STATUS"`
	RequestedAt     time.Time  `json:"requestedAt" db:"REQUESTED_AT"`
	TargetAt        time.Time  `json:"targetAt" db:"TARGET_AT"`
	DueAt           time.Time  `json:"dueAt" db:"DUE_AT"`
	CompletedAt     *time.Time `json:"completedAt" db:"COMPLETED_AT"`
	RetainedRecords string     `json:"retainedRecords" db:"RETAINED_RECORDS"`
	RetentionUntil  *time.Time `json:"retentionUntil" db:"RETENTION_UNTIL"`
}

type AccountDeletionQueueItem struct {
	NextAttemptAt       *time.Time `json:"nextAttemptAt" db:"NEXT_ATTEMPT_AT"`
	AutomationUpdatedAt *time.Time `json:"automationUpdatedAt" db:"AUTOMATION_UPDATED_AT"`
	ProcessingMode      string     `json:"processingMode" db:"PROCESSING_MODE"`
	AutoStage           string     `json:"autoStage" db:"AUTO_STAGE"`
	AutoCode            string     `json:"autoCode" db:"AUTO_CODE"`
	AccountDeletionReceipt
	UserSeq           *int   `json:"userSeq" db:"USR_SEQ"`
	EvidenceReference string `json:"evidenceReference" db:"EVIDENCE_REFERENCE"`
}

type AccountDeletionResolution struct {
	Action                  string `json:"action"`
	EvidenceReference       string `json:"evidenceReference"`
	ResultNotified          bool   `json:"resultNotified"`
	FilesErased             bool   `json:"filesErased"`
	BackupsErased           bool   `json:"backupsErased"`
	ExternalDataErased      bool   `json:"externalDataErased"`
	OtherIdentifiersChecked bool   `json:"otherIdentifiersChecked"`
	RetainedRecords         string `json:"retainedRecords"`
	RetentionUntil          string `json:"retentionUntil"`
}

type AccountDeletionFootprint struct {
	Table  string `json:"table"`
	Column string `json:"column"`
	Count  int64  `json:"count"`
}
