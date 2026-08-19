package model

import "strings"

type ExcelDonationRow struct {
	RowIndex        int    `json:"rowIndex"`
	DonorName       string `json:"donorName"`
	DonorCohort     string `json:"donorCohort"`
	DonorDepartment string `json:"donorDepartment"`
	DonorPhone      string `json:"donorPhone"`
	Amount          int64  `json:"amount"`
}

type DonationImportRowStatus string

const (
	DonationImportRowMatched   DonationImportRowStatus = "matched"
	DonationImportRowAmbiguous DonationImportRowStatus = "ambiguous"
	DonationImportRowUnmatched DonationImportRowStatus = "unmatched"
	DonationImportRowDuplicate DonationImportRowStatus = "duplicate"
)

type MemberCandidate struct {
	USRSeq int    `db:"USR_SEQ" json:"usrSeq"`
	Name   string `db:"USR_NAME" json:"name"`
}

type DonationImportMemberKey struct {
	Name   string
	Cohort string
	Phone  string
}

func NewDonationImportMemberKey(name, cohort, phone string) DonationImportMemberKey {
	return DonationImportMemberKey{
		Name:   strings.TrimSpace(name),
		Cohort: strings.TrimSpace(cohort),
		Phone:  NormalizePhoneNumber(phone).String(),
	}
}

type DonationImportPreviewRow struct {
	ExcelDonationRow
	DonationDate  string                  `json:"donationDate"`
	Status        DonationImportRowStatus `json:"status"`
	MatchedUsrSeq *int                    `json:"matchedUsrSeq"`
	MatchedName   string                  `json:"matchedName"`
	Note          string                  `json:"note"`
	PreviewToken  string                  `json:"previewToken"`
}

type DonationImportPreviewResult struct {
	Rows           []DonationImportPreviewRow `json:"rows"`
	MatchedCount   int                        `json:"matchedCount"`
	AmbiguousCount int                        `json:"ambiguousCount"`
	UnmatchedCount int                        `json:"unmatchedCount"`
	DuplicateCount int                        `json:"duplicateCount"`
}

type DonationImportCommitRow struct {
	ExcelDonationRow
	AccountUsrSeq *int   `json:"accountUsrSeq"`
	PreviewToken  string `json:"previewToken"`
}

type DonationImportCommitRequest struct {
	DonationDate string                    `json:"donationDate"`
	Rows         []DonationImportCommitRow `json:"rows"`
}

// ImportedDonationRow combines the position-based Excel values with the
// batch-level donation date supplied by the administrator.
type ImportedDonationRow struct {
	ExcelDonationRow
	DonationDate string
}

type ImportedDonationOrder struct {
	ImportedDonationRow
	AccountUsrSeq *int
}

type DonationImportCommitRowResult struct {
	RowIndex     int    `json:"rowIndex"`
	Success      bool   `json:"success"`
	OrderSeq     *int64 `json:"orderSeq"`
	ErrorMessage string `json:"errorMessage"`
}

type DonationImportCommitResult struct {
	Rows []DonationImportCommitRowResult `json:"rows"`
}

type DonationImportRowError struct {
	RowIndex int    `json:"rowIndex"`
	Field    string `json:"field"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

type DonationImportErrorResponse struct {
	Code    string                   `json:"code"`
	Message string                   `json:"message"`
	Errors  []DonationImportRowError `json:"errors"`
}
