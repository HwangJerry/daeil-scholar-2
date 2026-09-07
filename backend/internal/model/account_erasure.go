// account_erasure.go — Automatic erasure contracts and narrowly scoped blockers.
package model

import "time"

type ErasureBlocked struct{ Code string }

func (e *ErasureBlocked) Error() string { return e.Code }

type ErasureWork struct {
	ContextRetentions []DonationRetentionDecision `db:"-" json:"-"`
	RequestID         int64                       `db:"REQUEST_ID" json:"requestId"`
	UserSeq           int                         `db:"USR_SEQ" json:"userSeq"`
	ExternalEvidence  string                      `db:"EXTERNAL_EVIDENCE" json:"-"`
	Stage             string                      `db:"STAGE" json:"stage"`
}
type ErasureFile struct {
	ID  int64  `db:"ID"`
	URL string `db:"URL_PATH"`
}

// Sent only to the configured trusted erasure processor, never ordinary logs.
type ErasureExternalSubject struct {
	ExternalFileURLs []string                    `json:"externalFileUrls,omitempty"`
	RequiredTargets  []string                    `json:"requiredTargets,omitempty"`
	Retentions       []DonationRetentionDecision `json:"donationRetentions"`
	RequestID        int64                       `json:"requestId"`
	UserSeq          int                         `json:"userSeq"`
	Login            string                      `json:"login"`
	Email            string                      `json:"email"`
	Phone            string                      `json:"phone"`
	ProviderSubjects []string                    `json:"providerSubjects"`
}
type DonationRetentionDecision struct {
	SourceFingerprint string     `db:"SOURCE_FINGERPRINT" json:"sourceFingerprint"`
	OrderID           int        `db:"O_SEQ" json:"orderId"`
	Basis             string     `db:"BASIS" json:"basis"`
	BasisDate         *time.Time `db:"BASIS_DATE" json:"basisDate"`
	Until             *time.Time `db:"RETAIN_UNTIL" json:"retainUntil"`
	Evidence          string     `db:"EVIDENCE_REFERENCE" json:"evidenceReference"`
}
