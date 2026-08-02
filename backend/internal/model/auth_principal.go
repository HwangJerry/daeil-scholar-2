// auth_principal.go — Canonical authorization state included in authenticated sessions.
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

func (s VerificationStatus) Valid() bool {
	switch s {
	case VerificationUnsubmitted,
		VerificationPending,
		VerificationRejected,
		VerificationApproved,
		VerificationReapprovalPending:
		return true
	default:
		return false
	}
}

func (s VerificationStatus) AfterAcademicSubmission(academicChanged bool) VerificationStatus {
	if s == VerificationUnsubmitted || s == VerificationRejected {
		return VerificationPending
	}
	if s == VerificationApproved && academicChanged {
		return VerificationReapprovalPending
	}
	return s
}

type AdminRole string

const (
	AdminRoleRoot     AdminRole = "root"
	AdminRoleOperator AdminRole = "operator"
)

func (r AdminRole) Valid() bool {
	return r == AdminRoleRoot || r == AdminRoleOperator
}

type AlumniVerification struct {
	Status          VerificationStatus `json:"status"`
	GraduationYear  *int               `json:"graduationYear"`
	Cohort          *string            `json:"cohort"`
	Department      *string            `json:"department"`
	RejectionReason *string            `json:"rejectionReason"`
	SubmittedAt     *time.Time         `json:"submittedAt"`
	ReviewedAt      *time.Time         `json:"reviewedAt"`
}
