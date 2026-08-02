// login_eligibility_test.go — Verifies account lifecycle gates independently of alumni approval.
package service

import (
	"errors"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestLoginEligibilityIgnoresAlumniVerificationState(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr error
	}{
		{name: "legacy rejected remains session eligible", status: "BAA"},
		{name: "pending remains session eligible", status: "BBB"},
		{name: "approved remains session eligible", status: "CCC"},
		{name: "legacy admin remains session eligible", status: "ZZZ"},
		{name: "withdrawn is denied", status: "AAA", wantErr: ErrLoginWithdrawn},
		{name: "unknown suspended status is denied", status: "DDD", wantErr: ErrLoginSuspended},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := (LoginEligibilityPolicy{}).EnsureLoginAllowed(&model.User{USRStatus: test.status})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("EnsureLoginAllowed(%q) error = %v, want %v", test.status, err, test.wantErr)
			}
		})
	}
}
