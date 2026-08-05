// auth_principal.go — Canonical authorization state included in mobile sessions.
package model

import "time"

type VerificationStatus string

const (
	VerificationUnsubmitted       VerificationStatus = "unsubmitted"
	VerificationPending           VerificationStatus = "pending"
	VerificationRejected          VerificationStatus = "rejected"
	VerificationApproved          VerificationStatus = "approved"
	VerificationReapprovalPending VerificationStatus = "reapproval_pending"
)

type AdminRole string

type AlumniVerification struct {
	Status          VerificationStatus `json:"status"`
	GraduationYear  *int               `json:"graduationYear"`
	Cohort          *string            `json:"cohort"`
	Department      *string            `json:"department"`
	RejectionReason *string            `json:"rejectionReason"`
	SubmittedAt     *time.Time         `json:"submittedAt"`
	ReviewedAt      *time.Time         `json:"reviewedAt"`
}
