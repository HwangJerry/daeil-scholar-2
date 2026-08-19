package model

type ExcelDonationRow struct {
	RowIndex      int    `json:"rowIndex"`
	DonorName     string `json:"donorName"`
	DonorPhone    string `json:"donorPhone"`
	Amount        int64  `json:"amount"`
	DonationDate  string `json:"donationDate"`
	TransactionNo string `json:"transactionNo"`
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

type DonationImportPreviewRow struct {
	ExcelDonationRow
	Status        DonationImportRowStatus `json:"status"`
	MatchedUsrSeq *int                    `json:"matchedUsrSeq"`
	MatchedName   string                  `json:"matchedName"`
	Note          string                  `json:"note"`
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
	AccountUsrSeq *int `json:"accountUsrSeq"`
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
