package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const approvedCandidateManifestSHA256 = "979a1fe9f4a2416bbb4cdada329fe7d6a1a163d5d403d522656cea954770029a"

func TestMigrationRunnerApprovesEveryFutureMigrationIncluding055(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "migrations", "testdata", "canonical_identity_candidate_lineage.sha256")
	if _, err := os.ReadFile(manifestPath); err != nil {
		t.Fatalf("read candidate manifest: %v", err)
	}
	envExamplePath := filepath.Join("..", "..", ".env.example")
	envExample, err := os.ReadFile(envExamplePath)
	if err != nil {
		t.Fatalf("read backend environment example: %v", err)
	}
	exampleApproval := "# CANONICAL_CANDIDATE_MANIFEST_SHA256=" + approvedCandidateManifestSHA256
	if !strings.Contains(string(envExample), exampleApproval) {
		t.Fatalf("backend environment example is missing the reviewed candidate manifest approval")
	}

	t.Setenv("CANONICAL_CANDIDATE_MANIFEST_SHA256", approvedCandidateManifestSHA256)
	command := exec.Command("bash", filepath.Join("..", "..", "..", "migrate.sh"), "--check-source-approval")
	command.Env = os.Environ()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("migration source approval failed: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "MIGRATION_SOURCE_APPROVAL=PASS future_migrations=16" {
		t.Fatalf("unexpected source approval output: %q", output)
	}
}
