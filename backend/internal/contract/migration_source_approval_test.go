package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const approvedCandidateManifestSHA256 = "3521ead85cab31e59ec627b1be35f0c09c0c3c67079b0639223a2561b105002e"

func TestMigrationRunnerApprovesEveryFutureMigrationIncluding063(t *testing.T) {
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
	if strings.TrimSpace(string(output)) != "MIGRATION_SOURCE_APPROVAL=PASS future_migrations=28" {
		t.Fatalf("unexpected source approval output: %q", output)
	}
}
