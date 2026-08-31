package contract

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAccountLifecycleMigrationDefinesCanonicalPhoneAuthority(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(filename), "..", "..", "migrations", "044_enforce_account_lifecycle_invariants.sql")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sql := string(contents)
	for _, required := range []string{
		"CREATE TABLE AUTH_PHONE_CLAIM",
		"PRIMARY KEY (CANONICAL_PHONE)",
		"UNIQUE KEY UQ_AUTH_PHONE_CLAIM_ACCOUNT (ACCOUNT_ID)",
		"_044_phone_claim_conflicts",
		"COALESCE(USR_STATUS, '') <> 'AAA'",
		"CHAR_LENGTH(CANONICAL_PHONE) NOT BETWEEN 7 AND 15",
		"UQ_SRO_OPEN_ACTION",
		"_044_open_outbox_conflicts",
		"_044_build_phone_claim_source",
		"ASCII(candidate_character) BETWEEN ASCII('0') AND ASCII('9')",
		"deployed.ENGINE = 'InnoDB'",
		"ALUMNI_SOCIAL_REVOCATION_OUTBOX",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 044 missing %q", required)
		}
	}
	if count := strings.Count(sql, "WHERE CHAR_LENGTH(CANONICAL_PHONE) > 0"); count != 2 {
		t.Fatalf("migration 044 must exclude empty canonical phones from conflict detection and claim insertion, found %d filters", count)
	}
}
