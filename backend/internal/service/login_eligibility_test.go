package service

import (
	"errors"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestLoginEligibilityPolicy(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr error
	}{
		{name: "approved", status: "CCC"},
		{name: "operator", status: "ZZZ"},
		{name: "pending", status: "BBB", wantErr: ErrLoginPending},
		{name: "withdrawn", status: "AAA", wantErr: ErrLoginWithdrawn},
		{name: "suspended or unknown", status: "DDD", wantErr: ErrLoginSuspended},
	}
	policy := LoginEligibilityPolicy{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := policy.EnsureLoginAllowed(&model.User{USRStatus: test.status})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("EnsureLoginAllowed(%q) error = %v, want %v", test.status, err, test.wantErr)
			}
		})
	}
}
