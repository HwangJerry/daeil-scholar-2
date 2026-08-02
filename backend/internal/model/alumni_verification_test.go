package model

import "testing"

func TestVerificationStatusAfterSubmissionMovesUnsubmittedToPending(t *testing.T) {
	status := VerificationUnsubmitted.AfterAcademicSubmission(true)
	if status != VerificationPending {
		t.Fatalf("status = %q, want %q", status, VerificationPending)
	}
}

func TestVerificationStatusAfterSubmissionFollowsCanonicalTransitionTable(t *testing.T) {
	tests := []struct {
		name            string
		current         VerificationStatus
		academicChanged bool
		want            VerificationStatus
	}{
		{name: "rejected resubmission", current: VerificationRejected, academicChanged: true, want: VerificationPending},
		{name: "approved unchanged", current: VerificationApproved, academicChanged: false, want: VerificationApproved},
		{name: "approved academic change", current: VerificationApproved, academicChanged: true, want: VerificationReapprovalPending},
		{name: "pending correction", current: VerificationPending, academicChanged: true, want: VerificationPending},
		{name: "reapproval correction", current: VerificationReapprovalPending, academicChanged: true, want: VerificationReapprovalPending},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := test.current.AfterAcademicSubmission(test.academicChanged)
			if status != test.want {
				t.Fatalf("status = %q, want %q", status, test.want)
			}
		})
	}
}
