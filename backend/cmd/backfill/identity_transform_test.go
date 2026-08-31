// Identity transform tests — verifies deterministic mappings, fingerprints, and conflict classification.
package main

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TestMapLegacyAccountStatus(t *testing.T) {
	tests := []struct {
		legacy string
		want   canonicalAccountState
	}{
		{legacy: "BAA", want: canonicalAccountState{Status: accountStatusActive}},
		{legacy: "BBB", want: canonicalAccountState{Status: accountStatusActive}},
		{legacy: "CCC", want: canonicalAccountState{Status: accountStatusActive}},
		{legacy: "ZZZ", want: canonicalAccountState{Status: accountStatusActive}},
		{legacy: "AAA", want: canonicalAccountState{Status: accountStatusWithdrawn, WithdrawnAt: true}},
		{legacy: "DDD", want: canonicalAccountState{Status: accountStatusSuspended, SuspendedAt: true}},
		{legacy: "", want: canonicalAccountState{Status: accountStatusSuspended, SuspendedAt: true}},
		{legacy: "unknown", want: canonicalAccountState{Status: accountStatusSuspended, SuspendedAt: true}},
	}
	for _, test := range tests {
		t.Run(test.legacy, func(t *testing.T) {
			if got := mapLegacyAccountStatus(test.legacy); got != test.want {
				t.Fatalf("mapLegacyAccountStatus(%q) = %+v, want %+v", test.legacy, got, test.want)
			}
		})
	}
}

func TestBackfillPasswordIdentitiesSkipsWithdrawnMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.ExpectBegin()
	tx, err := sqlxDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTxx: %v", err)
	}

	const runID = "withdrawn-password-skip"
	mock.ExpectExec(`INSERT INTO AUTH_IDENTITY_MIGRATION_JOURNAL`).
		WithArgs(runID, passwordIdentitiesStep).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE AUTH_IDENTITY_MIGRATION_JOURNAL`).
		WithArgs(runID, passwordIdentitiesStep).
		WillReturnResult(sqlmock.NewResult(0, 1))

	stats := identityBackfillStats{}
	members := []identityMemberRow{{
		AccountID:    42,
		Username:     "01012345678",
		Email:        "member@example.com",
		PasswordHash: "*2470C0C06DEE42FD1618BB99005ADCA2EC9D1E19",
		LegacyStatus: "AAA",
	}}
	if err := backfillPasswordIdentities(ctx, tx, runID, members, &stats); err != nil {
		t.Fatalf("backfillPasswordIdentities: %v", err)
	}
	if stats.PasswordIdentitiesCreated != 0 || stats.PasswordCredentialsCreated != 0 || stats.ConflictCount != 0 {
		t.Fatalf("withdrawn member stats = %+v, want no password inserts or conflicts", stats)
	}

	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestSourceFingerprint(t *testing.T) {
	const want = "3e94de3844055899bbe2b936ba7f61d024e390dc510e072725cc241c88e84cfe"
	if got := sourceFingerprint(7, 42); got != want {
		t.Fatalf("sourceFingerprint(7, 42) = %q, want %q", got, want)
	}
	if got := sourceFingerprint(7, 43); got == want {
		t.Fatal("source fingerprint did not change with max USR_SEQ")
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := normalizedEmail("  MEMBER@Example.COM "); got == nil || *got != "member@example.com" {
		t.Fatalf("normalizedEmail() = %#v", got)
	}
	if got := normalizedEmail("  "); got != nil {
		t.Fatalf("normalizedEmail(blank) = %#v, want nil", got)
	}
}

func TestLegacyProviderAndStatusMappings(t *testing.T) {
	if provider, ok := mapLegacySocialProvider("KT"); !ok || provider != identityProviderKakao {
		t.Fatalf("KT mapping = %q, %t", provider, ok)
	}
	if provider, ok := mapLegacySocialProvider("AP"); !ok || provider != identityProviderApple {
		t.Fatalf("AP mapping = %q, %t", provider, ok)
	}
	if _, ok := mapLegacySocialProvider("KAKAO"); ok {
		t.Fatal("non-legacy social provider unexpectedly accepted")
	}
	if status, ok := mapLegacySocialStatus("Y"); !ok || status != identityStatusActive {
		t.Fatalf("Y mapping = %q, %t", status, ok)
	}
	if status, ok := mapLegacySocialStatus("INACTIVE"); !ok || status != identityStatusDisabled {
		t.Fatalf("INACTIVE mapping = %q, %t", status, ok)
	}
}

func TestMysqlNativePasswordValidation(t *testing.T) {
	valid := "*2470C0C06DEE42FD1618BB99005ADCA2EC9D1E19"
	if !isMysqlNativePasswordHash(valid) {
		t.Fatal("valid MySQL native password hash rejected")
	}
	for _, invalid := range []string{"", "plaintext", "*not-hex", "UNREADABLE_ALGORITHM_TAG:fixture"} {
		if isMysqlNativePasswordHash(invalid) {
			t.Fatalf("invalid password hash %q accepted", invalid)
		}
	}
}

func TestConflictCountingAndClassification(t *testing.T) {
	count := 0
	count = incrementConflictCount(count, false)
	count = incrementConflictCount(count, true)
	count = incrementConflictCount(count, true)
	if count != 2 {
		t.Fatalf("conflict count = %d, want 2", count)
	}

	reason, ok := classifyUniqueConstraintError(&mysql.MySQLError{
		Number:  duplicateEntryErrorNumber,
		Message: "Duplicate entry for key 'UQ_AUTH_IDENTITY_NORMALIZED_EMAIL'",
	}, "duplicate_identity")
	if !ok || reason != "duplicate_normalized_email" {
		t.Fatalf("classification = %q, %t", reason, ok)
	}
	if _, ok := classifyUniqueConstraintError(errors.New("connection lost"), "duplicate_identity"); ok {
		t.Fatal("non-MySQL error classified as a unique conflict")
	}
}

func TestIdentityKeyValidation(t *testing.T) {
	if !isValidIdentityKey("ascii-subject") {
		t.Fatal("valid ASCII identity key rejected")
	}
	for _, invalid := range []string{"", "   ", "한글-subject"} {
		if isValidIdentityKey(invalid) {
			t.Fatalf("invalid identity key %q accepted", invalid)
		}
	}
}
