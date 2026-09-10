package model

import "testing"

func TestVerificationStatusAfterSubmissionMovesUnsubmittedToPending(t *testing.T) {
	status := VerificationUnsubmitted.AfterAcademicSubmission()
	if status != VerificationPending {
		t.Fatalf("status = %q, want %q", status, VerificationPending)
	}
}

func TestVerificationStatusAfterSubmissionFollowsCanonicalTransitionTable(t *testing.T) {
	tests := []struct {
		name    string
		current VerificationStatus
		want    VerificationStatus
	}{
		{name: "rejected resubmission", current: VerificationRejected, want: VerificationPending},
		{name: "approved unchanged", current: VerificationApproved, want: VerificationApproved},
		{name: "approved academic change", current: VerificationApproved, want: VerificationApproved},
		{name: "pending correction", current: VerificationPending, want: VerificationPending},
		{name: "reapproval correction", current: VerificationReapprovalPending, want: VerificationApproved},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := test.current.AfterAcademicSubmission()
			if status != test.want {
				t.Fatalf("status = %q, want %q", status, test.want)
			}
		})
	}
}
