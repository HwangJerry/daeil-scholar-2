package repository

import (
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestSocialIdentityConflictRequiresProviderSubjectConstraint(t *testing.T) {
	for _, test := range []struct {
		key      string
		identity bool
	}{
		{"UK_PROVIDER_SUBJECT", true},
		{"UQ_AUTH_IDENTITY_PROVIDER_SUBJECT", true},
		{"AUTH_IDENTITY.UQ_AUTH_IDENTITY_PROVIDER_SUBJECT", true},
		{"UQ_AUTH_IDENTITY_NORMALIZED_EMAIL", false},
		{"UK_USR_PROVIDER", false},
		{"PRIMARY", false},
	} {
		t.Run(test.key, func(t *testing.T) {
			cause := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'fixture' for key '" + test.key + "'"}
			err := classifySocialIdentityInsertError(cause)
			if errors.Is(err, ErrSocialIdentityAlreadyLinked) != test.identity {
				t.Fatalf("identity conflict = %v, want %v: %v", errors.Is(err, ErrSocialIdentityAlreadyLinked), test.identity, err)
			}
			if !errors.Is(err, cause) {
				t.Fatal("database cause must be preserved")
			}
		})
	}
}
