package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSocialAuthMigrationContracts(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "testdata", "current-branch-8dcba0b", "028_social_auth_security.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	required := []string{
		"ADD COLUMN NMS_EMAIL",
		"UNIQUE KEY UK_USR_PROVIDER (USR_SEQ, NMS_GATE)",
		"ALUMNI_MOBILE_REFRESH_TOKEN",
		"ALUMNI_APPLE_NONCE_CHALLENGE",
		"ALUMNI_APPLE_CODE_REPLAY",
		"ALUMNI_SOCIAL_CREDENTIAL",
		"ALUMNI_SOCIAL_REVOCATION_OUTBOX",
		"UNIQUE KEY UK_GATE_ID",
	}
	// The provider-subject unique key is defined in migration 007 and must remain.
	migration007, err := os.ReadFile(filepath.Join("..", "..", "migrations", "007_create_member_social_table.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql += string(migration007)
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration contract missing %q", fragment)
		}
	}
}

func TestCanonicalPhoneMigrationContracts(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", "testdata", "current-branch-8dcba0b", "029_canonical_member_phone.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	required := []string{
		"HAVING COUNT(*) > 1",
		"UPDATE WEO_MEMBER",
		"REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '')",
		"CREATE INDEX IDX_WEO_MEMBER_PHONE ON WEO_MEMBER (USR_PHONE)",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("canonical phone migration contract missing %q", fragment)
		}
	}
}
