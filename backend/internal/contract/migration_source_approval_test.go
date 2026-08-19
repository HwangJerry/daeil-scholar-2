package contract

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationRunnerApprovesEveryFutureMigrationIncluding049(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "migrations", "testdata", "canonical_identity_candidate_lineage.sha256")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read candidate manifest: %v", err)
	}

	command := exec.Command("bash", filepath.Join("..", "..", "..", "migrate.sh"), "--check-source-approval")
	command.Env = append(os.Environ(), fmt.Sprintf("CANONICAL_CANDIDATE_MANIFEST_SHA256=%x", sha256.Sum256(manifest)))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("migration source approval failed: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "MIGRATION_SOURCE_APPROVAL=PASS future_migrations=10" {
		t.Fatalf("unexpected source approval output: %q", output)
	}
}
