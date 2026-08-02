package model

import "time"

// AlumniVerificationSubmissionRequest is the complete academic record submitted
// for initial verification, correction, or reapproval.
type AlumniVerificationSubmissionRequest struct {
	GraduationYear int    `json:"graduationYear"`
	Cohort         string `json:"cohort"`
	Department     string `json:"department"`
}

type AdminAlumniVerification struct {
	UserSeq         int                `db:"USR_SEQ" json:"userSeq"`
	UserName        string             `db:"USR_NAME" json:"userName"`
	Status          VerificationStatus `db:"STATUS" json:"status"`
	GraduationYear  *int               `db:"GRADUATION_YEAR" json:"graduationYear"`
	Cohort          *string            `db:"COHORT" json:"cohort"`
	Department      *string            `db:"DEPARTMENT" json:"department"`
	RejectionReason *string            `db:"REJECTION_REASON" json:"rejectionReason"`
	SubmittedAt     *time.Time         `db:"SUBMITTED_AT" json:"submittedAt"`
	ReviewedAt      *time.Time         `db:"REVIEWED_AT" json:"reviewedAt"`
	UpdatedAt       time.Time          `db:"UPDATED_AT" json:"updatedAt"`
}
