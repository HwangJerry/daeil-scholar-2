// canonical_identity_migration_harness_integration_test.go — Executes the pinned exact-engine harness on demand.
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

func TestCanonicalIdentityMigrationHarnessExpectedCurrentRed(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--expect-current-red")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("migration harness failed: %v\n%s", err, output)
	}

	for _, expected := range []string{
		"CURRENT_BRANCH_028_TO_036=EXPECTED_RED code=ERROR_1054 residual_procedures=2",
		"AUTHORITATIVE_GIT_UPGRADE=EXPECTED_RED missing canonical schema history=12",
		"DUPLICATE_PROVIDER_SUBJECT=EXPECTED_RED rejected_before_table_alter residual_procedures=3",
		"DUPLICATE_NORMALIZED_EMAIL=EXPECTED_RED preflight_consumer_missing",
		"ORPHAN_SOCIAL_ROW=EXPECTED_RED preflight_consumer_missing",
		"MALFORMED_IDENTITY=EXPECTED_RED preflight_consumer_missing",
		"UNREADABLE_ALGORITHM_TAG=EXPECTED_RED preflight_consumer_missing",
		"HISTORY_NUMBER_COLLISION=EXPECTED_RED",
		"HISTORY_DIGEST_DRIFT=EXPECTED_RED",
		"HISTORY_MISSING_035=EXPECTED_RED",
		"HISTORY_ROW_SCHEMA_ABSENT=EXPECTED_RED",
		"SCHEMA_PRESENT_HISTORY_ABSENT=EXPECTED_RED",
		"HARNESS_SELF_TEST=PASS",
	} {
		if !strings.Contains(string(output), expected) {
			t.Fatalf("migration harness output is missing %q\n%s", expected, output)
		}
	}
	if strings.Contains(string(output), "CONFLICT_FIXTURE=PASS") {
		t.Fatalf("migration harness collapsed independent conflict REDs into a false PASS\n%s", output)
	}
}

func TestCanonicalIdentitySourceLineageCheck(t *testing.T) {
	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--check-source-lineage")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("source-lineage check failed: %v\n%s", err, output)
	}

	const expected = `AUTHORITATIVE_TESTDATA_LINEAGE=PASS entries=12
AUTHORITATIVE_SOURCE_LINEAGE=PASS`
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected source-lineage output: %q", output)
	}
}

func TestCanonicalIdentityCandidateRangeCheck(t *testing.T) {
	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := approvedCurrentCandidateCommand(t, script, "--check-candidate-range")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("candidate-range check failed: %v\n%s", err, output)
	}

	const expected = "CANONICAL_CANDIDATE_RANGE=PASS maintenance_excluded=043"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected candidate-range output: %q", output)
	}
}

func TestCanonicalIdentityPostconditionNegativeControl(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script, _, _ := copyCandidatePacket(t)
	command := approvedCandidateCommand(t, script, "--self-test-postconditions")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("postcondition negative control failed: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "POSTCONDITION_NEGATIVE_CONTROL=PASS" {
		t.Fatalf("unexpected postcondition negative-control output: %q", output)
	}
}

func TestCanonicalIdentityCleanupFailureOverridesSuccess(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-postconditions")
	command.Env = append(os.Environ(), "HERMES_TEST_CLEANUP_COMMAND_FAILURE=1")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("cleanup failure unexpectedly returned success\n%s", output)
	}
	if exitError, ok := err.(*exec.ExitError); !ok || exitError.ExitCode() != 125 {
		t.Fatalf("cleanup failure exit = %v, want 125\n%s", err, output)
	}
	if strings.Contains(string(output), "POSTCONDITION_NEGATIVE_CONTROL=PASS") {
		t.Fatalf("cleanup failure emitted a PASS marker\n%s", output)
	}
}

func TestMigrationRunnerCollisionExpressionIsSQLModeIndependent(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-runner-sql-mode")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("runner SQL-mode collision probe failed: %v\n%s", err, output)
	}
	const expected = "RUNNER_COLLISION_SQL_MODE=PASS default=1 no_backslash_escapes=1 paths=normal,seed"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected runner SQL-mode collision output: %q", output)
	}
}

func TestMigrationRunnerExactMariaDBHistoryAuthority(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-migration-runner")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("exact MariaDB migration-runner probe failed: %v\n%s", err, output)
	}
	for _, expected := range []string{
		"RUNNER_HISTORY_AUTHORITY=PASS cases=fresh,reconciled,started_failure,schema_present_history_absent,filename_only,wrong_shape,prefix_primary_key,extra_column,extra_index,malformed_digest,source_collision",
		"RUNNER_NUMBER_COLLISION=PASS modes=default,no_backslash_escapes paths=normal,seed",
		"RUNNER_CANDIDATE_APPROVAL=PASS cases=missing_approval,env_self_authorization,wrong_approval,approved,co_drift,extra_source",
	} {
		if !strings.Contains(string(output), expected) {
			t.Fatalf("migration-runner probe output is missing %q\n%s", expected, output)
		}
	}
}

func TestProductionLineageReconciliationExactMariaDB(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-production-lineage-reconciliation")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("production lineage reconciliation probe failed: %v\n%s", err, output)
	}
	const expected = "PRODUCTION_LINEAGE_RECONCILIATION=PASS cases=missing_approval,wrong_approval,wrong_filename,partial_resume,partial_wrong_digest,legacy,target history=39 journal=39 runner_noop=39"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected production lineage reconciliation output: %q", output)
	}
}

func TestCanonicalIdentityT03TransactionBoundaryRollback(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t03-transaction-boundary")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T03 transaction-boundary probe failed: %v\n%s", err, output)
	}
	const expected = `T03_TRANSACTION_BOUNDARY=EXPECTED_RED before=1:0
T03_TRANSACTION_BOUNDARY=PASS before=1:0 after=0:0 engine=InnoDB`
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected T03 transaction-boundary output: %q", output)
	}
}

func TestCanonicalIdentityT03RejectsUnexpectedSourceEngine(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t03-unexpected-engine")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T03 unexpected-engine probe failed: %v\n%s", err, output)
	}
	const expected = "T03_UNEXPECTED_ENGINE=PASS source=MEMORY history=0 procedures=0"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected T03 engine-guard output: %q", output)
	}
}

func TestCanonicalIdentityT03FailsClosedAfterDurableDDLInterruption(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t03-target-resume")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T03 target-resume probe failed: %v\n%s", err, output)
	}
	const expected = "T03_STARTED_FAIL_CLOSED=PASS preflight=bound socket=reject unbound=reject cause=injected_interruption engine=InnoDB history=0 journal=STARTED lock=1 replay=refused recovery=verified-backup-restore"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected T03 target-resume output: %q", output)
	}
}

func TestCanonicalIdentityT03BoundRunnerAppliesExactMigration(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t03-bound-apply")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T03 bound apply probe failed: %v\n%s", err, output)
	}
	const expected = "T03_BOUND_APPLY=PASS preflight=bound engine=InnoDB history=1 journal=APPLIED lock=0 rollback=0:0 procedures=0"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected T03 bound apply output: %q", output)
	}
}

func TestCanonicalIdentityT03CandidateManifestPinsExactMigration(t *testing.T) {
	migrations := canonicalMigrationDir()
	migrationPath := filepath.Join(migrations, "040_convert_auth_transaction_boundary_to_innodb.sql")
	migration, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read T03 migration: %v", err)
	}
	wantDigest := fmt.Sprintf("%x", sha256.Sum256(migration))
	manifestPath := filepath.Join(migrations, "testdata", "canonical_identity_candidate_lineage.sha256")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read T03 candidate manifest: %v", err)
	}
	matches := 0
	for _, line := range strings.Split(strings.TrimSpace(string(manifest)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != "040_convert_auth_transaction_boundary_to_innodb.sql" {
			continue
		}
		matches++
		if fields[0] != wantDigest {
			t.Fatalf("T03 candidate manifest pins the wrong migration 040 digest")
		}
	}
	if matches != 1 {
		t.Fatalf("T03 candidate manifest has %d migration 040 entries, want 1", matches)
	}
}

func TestCanonicalIdentityT03PreflightReportsConversionCapacity(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t03-preflight")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T03 preflight probe failed: %v\n%s", err, output)
	}
	fields := strings.Fields(strings.TrimSpace(string(output)))
	for _, expected := range []string{
		"T03_AUTH_ENGINE_PREFLIGHT=PASS",
		"engine=MyISAM",
		"state=source",
		"rows=2",
		"unsupported_spatial_indexes=0",
		"headroom=pass",
		"writer_freeze=verified",
		"writer_masks=verified",
		"database_locality=unix-socket",
		"snapshot=locked",
		"arithmetic=bounded",
		"stderr=redacted",
		"options=stable-fd-copy",
		"execution_binding=pending-controller",
	} {
		if !containsField(fields, expected) {
			t.Fatalf("T03 preflight output is missing %q: %q", expected, output)
		}
	}
}

func TestCanonicalIdentityT03PreflightNegativeControls(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t03-preflight-negative-controls")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T03 preflight negative controls failed: %v\n%s", err, output)
	}
	const expected = "T03_PREFLIGHT_NEGATIVE_CONTROLS=PASS ancestry=reject mask=reject active=reject locality=reject overflow=reject stderr=redacted"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected T03 preflight negative-control output: %q", output)
	}
}

func TestCanonicalIdentityT03ConversionPreservesMemberTable(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t03-preservation")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T03 preservation probe failed: %v\n%s", err, output)
	}
	const expected = "T03_CONVERSION_PRESERVATION=PASS rows=2 columns=same indexes=same triggers=same auto_increment=same checksum=same rerun=pass negative_control=detected"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected T03 preservation output: %q", output)
	}
}

func TestCanonicalIdentityT04IdentityCardinality(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t04-identity-cardinality")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T04 identity-cardinality probe failed: %v\n%s", err, output)
	}
	const expected = "T04_IDENTITY_CARDINALITY=PASS same_account_provider=2 global_subject=reject nullable_email=2 verified_email=reject engine=InnoDB account_provider_unique=absent"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected T04 identity-cardinality output: %q", output)
	}
}

func TestCanonicalIdentityT04CredentialAndTokenStorageBoundaries(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t04-credential-token-boundaries")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T04 credential/token-boundary probe failed: %v\n%s", err, output)
	}
	const expected = "T04_CREDENTIAL_TOKEN_BOUNDARIES=PASS social_without_credentials=1 password=provider-bound provider_secret=exactly-one-provider-bound-owner verification=hash-only continuation=hash-only provider_secret_in_continuation=absent engines=InnoDB"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected T04 credential/token-boundary output: %q", output)
	}
}

func TestCanonicalIdentityT04ConsentSessionAndRevokeOutbox(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t04-consent-session-outbox")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T04 consent/session/outbox probe failed: %v\n%s", err, output)
	}
	const expected = "T04_CONSENT_SESSION_OUTBOX=PASS consent_versions=3 explicit_optional_false=1 session_identity_account_provider=bound session_generation=1 outbox_identity_provider_credential=bound outbox_idempotency=reject engines=InnoDB"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected T04 consent/session/outbox output: %q", output)
	}
}

func TestCanonicalIdentityT04AdditiveCutoverPreparation(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t04-additive-preparation")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T04 additive-preparation probe failed: %v\n%s", err, output)
	}
	const expected = "T04_ADDITIVE_PREPARATION=PASS migration_tables=2 journal_step_unique=reject engines=InnoDB legacy_triggers=2 legacy_member_rows=5 history=15 destructive_sql=absent"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected T04 additive-preparation output: %q", output)
	}
}

func TestCanonicalIdentityT04MaintenanceFinalization(t *testing.T) {
	if os.Getenv("CANONICAL_IDENTITY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CANONICAL_IDENTITY_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := exec.Command(script, "--self-test-t04-maintenance-finalization")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("T04 maintenance-finalization probe failed: %v\n%s", err, output)
	}
	const expected = "T04_MAINTENANCE_FINALIZATION=PASS unverified=reject stale_verified=reject unjournaled=reject failure_procedures=0 triggers=0 legacy_member_rows=5 mobile_refresh=revoked canonical_family=revoked history=16 rerun=pass procedures=0"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("unexpected T04 maintenance-finalization output: %q", output)
	}
}

func containsField(fields []string, expected string) bool {
	for _, field := range fields {
		if field == expected {
			return true
		}
	}
	return false
}
