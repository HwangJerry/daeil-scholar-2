// hall_of_fame.go — Hall of Fame domain types: wall entries, public payloads, admin payloads, import diffs.
package model

import (
	"bytes"
	"encoding/json"
	"time"
)

type HallOfFameKind string

const (
	HallOfFameKindAlumni    HallOfFameKind = "alumni"
	HallOfFameKindCompanion HallOfFameKind = "companion"
	HallOfFameKindTeacher   HallOfFameKind = "teacher"
	HallOfFameKindGuest     HallOfFameKind = "guest"
	HallOfFameKindOrg       HallOfFameKind = "org"
)

// Valid reports whether the kind is one of the five stored ENUM values.
func (k HallOfFameKind) Valid() bool {
	switch k {
	case HallOfFameKindAlumni, HallOfFameKindCompanion, HallOfFameKindTeacher, HallOfFameKindGuest, HallOfFameKindOrg:
		return true
	}
	return false
}

const (
	HallOfFameSourceExcel  = "excel"
	HallOfFameSourceManual = "manual"
)

// HallOfFameTiersDescending lists every plaque tier in display order. The public
// response always carries each tier, so the app can keep its plaque structure.
var HallOfFameTiersDescending = []int{1000, 500, 300, 100}

// IsHallOfFameTier replaces the CHECK constraint MariaDB 10.1 does not enforce.
func IsHallOfFameTier(tier int) bool {
	for _, candidate := range HallOfFameTiersDescending {
		if candidate == tier {
			return true
		}
	}
	return false
}

// HallOfFameEntry is one stored row. NaturalKey and PhoneHash never leave the server.
type HallOfFameEntry struct {
	ID            int64          `db:"id" json:"id"`
	NaturalKey    string         `db:"natural_key" json:"-"`
	DisplayName   string         `db:"display_name" json:"displayName"`
	Kind          HallOfFameKind `db:"kind" json:"kind"`
	Cohort        *int           `db:"cohort" json:"cohort"`
	Department    *string        `db:"department" json:"department"`
	TierManWon    *int           `db:"tier_man_won" json:"tierManWon"`
	ContactName   *string        `db:"contact_name" json:"contactName"`
	RoleLabel     *string        `db:"role_label" json:"roleLabel"`
	PhoneHash     *string        `db:"phone_hash" json:"-"`
	ConsentPublic bool           `db:"consent_public" json:"consentPublic"`
	Visible       bool           `db:"visible" json:"visible"`
	Source        string         `db:"source" json:"source"`
	SourceBatchID *string        `db:"source_batch_id" json:"sourceBatchId"`
	CreatedAt     time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updatedAt"`
	DeletedAt     *time.Time     `db:"deleted_at" json:"-"`
}

// HallOfFamePublicEntry deliberately has no kind, phone, amount or count field.
type HallOfFamePublicEntry struct {
	DisplayName string `json:"displayName"`
	Label       string `json:"label"`
}

type HallOfFamePublicTier struct {
	TierManWon int                     `json:"tierManWon"`
	Entries    []HallOfFamePublicEntry `json:"entries"`
}

type HallOfFamePublicResponse struct {
	Tiers []HallOfFamePublicTier `json:"tiers"`
}

// HallOfFameAdminEntry is the operator view of a row, with the computed public label.
type HallOfFameAdminEntry struct {
	HallOfFameEntry
	Label string `json:"label"`
}

type HallOfFameAdminFilter struct {
	Tier    *int
	NoTier  bool
	Kind    string
	Query   string
	Visible *bool
	Page    int
	Size    int
}

// HallOfFameTierCount is an operator-only summary; the public API has no counts.
type HallOfFameTierCount struct {
	TierManWon *int `db:"tier_man_won" json:"tierManWon"`
	Count      int  `db:"entry_count" json:"count"`
}

type HallOfFameAdminList struct {
	Items      []HallOfFameAdminEntry `json:"items"`
	Total      int                    `json:"total"`
	Page       int                    `json:"page"`
	Size       int                    `json:"size"`
	TierCounts []HallOfFameTierCount  `json:"tierCounts"`
}

// HallOfFameCreateInput is a manually added row. Omitted toggles default to true.
type HallOfFameCreateInput struct {
	DisplayName   string         `json:"displayName"`
	Kind          HallOfFameKind `json:"kind"`
	Cohort        *int           `json:"cohort"`
	Department    *string        `json:"department"`
	TierManWon    *int           `json:"tierManWon"`
	ContactName   *string        `json:"contactName"`
	RoleLabel     *string        `json:"roleLabel"`
	ConsentPublic *bool          `json:"consentPublic"`
	Visible       *bool          `json:"visible"`
}

// OptionalField distinguishes "absent" from an explicit JSON null in PATCH bodies.
type OptionalField[T any] struct {
	Set   bool
	Value *T
}

func (f *OptionalField[T]) UnmarshalJSON(data []byte) error {
	f.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		f.Value = nil
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	f.Value = &value
	return nil
}

// HallOfFamePatchInput applies only the fields present in the request body.
type HallOfFamePatchInput struct {
	DisplayName   OptionalField[string]         `json:"displayName"`
	Kind          OptionalField[HallOfFameKind] `json:"kind"`
	Cohort        OptionalField[int]            `json:"cohort"`
	Department    OptionalField[string]         `json:"department"`
	TierManWon    OptionalField[int]            `json:"tierManWon"`
	ContactName   OptionalField[string]         `json:"contactName"`
	RoleLabel     OptionalField[string]         `json:"roleLabel"`
	ConsentPublic OptionalField[bool]           `json:"consentPublic"`
	Visible       OptionalField[bool]           `json:"visible"`
}

// HallOfFameImportRow is one normalized spreadsheet row. The raw phone is dropped
// at parse time; only its keyed hash survives, so the token and diff never hold it.
type HallOfFameImportRow struct {
	RowIndex    int            `json:"rowIndex"`
	NaturalKey  string         `json:"-"`
	PhoneHash   *string        `json:"-"`
	DisplayName string         `json:"displayName"`
	Kind        HallOfFameKind `json:"kind"`
	Cohort      *int           `json:"cohort"`
	Department  *string        `json:"department"`
	TierManWon  *int           `json:"tierManWon"`
	ContactName *string        `json:"contactName"`
	RoleLabel   *string        `json:"roleLabel"`
	Label       string         `json:"label"`
}

type HallOfFameFieldChange struct {
	Field  string      `json:"field"`
	Before interface{} `json:"before"`
	After  interface{} `json:"after"`
}

type HallOfFameUpdatedRow struct {
	ID          int64                   `json:"id"`
	RowIndex    int                     `json:"rowIndex"`
	DisplayName string                  `json:"displayName"`
	Changes     []HallOfFameFieldChange `json:"changes"`
}

type HallOfFameRemovalCandidate struct {
	ID          int64          `json:"id"`
	DisplayName string         `json:"displayName"`
	Kind        HallOfFameKind `json:"kind"`
	Cohort      *int           `json:"cohort"`
	Department  *string        `json:"department"`
	TierManWon  *int           `json:"tierManWon"`
}

// HallOfFameImportIssue messages never echo cell values, so a phone number typed
// into the wrong column cannot surface here. RowIndex is null for file-level issues.
type HallOfFameImportIssue struct {
	RowIndex *int   `json:"rowIndex"`
	Field    string `json:"field"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// HallOfFameImportPreview is the diff shown before commit. PreviewToken is empty
// while the file has blocking errors, so such a file cannot be committed.
type HallOfFameImportPreview struct {
	Added             []HallOfFameImportRow        `json:"added"`
	Updated           []HallOfFameUpdatedRow       `json:"updated"`
	UnchangedCount    int                          `json:"unchangedCount"`
	ManualSkipped     int                          `json:"manualSkippedCount"`
	RemovalCandidates []HallOfFameRemovalCandidate `json:"removalCandidates"`
	Warnings          []HallOfFameImportIssue      `json:"warnings"`
	Errors            []HallOfFameImportIssue      `json:"errors"`
	AlreadyCommitted  bool                         `json:"alreadyCommitted"`
	PreviewToken      string                       `json:"previewToken"`
}

type HallOfFameImportCommitRequest struct {
	PreviewToken string  `json:"previewToken"`
	RemoveIDs    []int64 `json:"removeIds"`
}

type HallOfFameImportCommitResult struct {
	BatchID        string `json:"batchId,omitempty"`
	AddedCount     int    `json:"addedCount"`
	UpdatedCount   int    `json:"updatedCount"`
	RemovedCount   int    `json:"removedCount"`
	UnchangedCount int    `json:"unchangedCount"`
	ManualSkipped  int    `json:"manualSkippedCount"`
}

// HallOfFameImportBatch records one committed file.
type HallOfFameImportBatch struct {
	ID          string
	FileSHA256  string
	UploadedBy  int
	SummaryJSON string
}
