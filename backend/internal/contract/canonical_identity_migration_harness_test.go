// canonical_identity_migration_harness_test.go — Validates the disposable MariaDB migration harness contract.
package contract

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalIdentityMigrationHarnessContract(t *testing.T) {
	migrationDir := canonicalMigrationDir()
	assertFileEquals(t,
		filepath.Join(migrationDir, "testdata", "mariadb-10.1.38.image"),
		"mariadb@sha256:e4e5e5e2fb7c089688ddb55cc5ef38c9acff4aeb0aa25375f92f0708795b7a1c\n",
	)
	assertFileContains(t, filepath.Join(migrationDir, "testdata", "canonical_identity_fresh_fixture.sql"), []string{
		"ENGINE=MyISAM", "WEO_MEMBER_SOCIAL", "ALUMNI_MOBILE_REFRESH_TOKEN", "MRT_REVOKED_AT",
	})
	assertFileContains(t, filepath.Join(migrationDir, "testdata", "authoritative-164788c", "kakao_auth_028_035_fixture.sql"), []string{
		"ENGINE=MyISAM", "CREATE TABLE WEO_MEMBER", "CREATE TABLE WEO_MEMBER_SOCIAL", "INSERT INTO WEO_MEMBER",
	})
	for _, fixture := range []string{
		"canonical_identity_conflict_duplicate_normalized_email.sql",
		"canonical_identity_conflict_duplicate_provider_subject.sql",
		"canonical_identity_conflict_orphan_social_row.sql",
		"canonical_identity_conflict_malformed_identity.sql",
		"canonical_identity_conflict_unreadable_algorithm.sql",
	} {
		if _, err := os.Stat(filepath.Join(migrationDir, "testdata", fixture)); err != nil {
			t.Fatalf("independent conflict fixture %s: %v", fixture, err)
		}
	}
	assertFileContains(t, filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh"), []string{
		"--expect-current-red", "--verify", "trap cleanup_trap EXIT", "docker rm -fv", "[REDACTED]", "MIXED_ENGINE_PARTIAL_COMMIT",
		"canonical_identity_schema_contract.sql", "canonical_identity_schema_contract.sha256", "CANONICAL_SCHEMA_CONTRACT_PASS",
	})
}

func TestMigrationHistoryFilenameUsesMariaDB101SafeCharset(t *testing.T) {
	const marker = "filename VARCHAR(255) CHARACTER SET ascii"
	for _, path := range []string{
		filepath.Join("..", "..", "..", "migrate.sh"),
		filepath.Join(canonicalMigrationDir(), "testdata", "canonical_identity_fresh_fixture.sql"),
		filepath.Join(canonicalMigrationDir(), "testdata", "canonical_identity_current_branch_pre_028_fixture.sql"),
	} {
		assertFileContains(t, path, []string{marker})
	}
}

func TestCanonicalIdentityHarnessRequiresFinalInitMarkerAndBoundedReadiness(t *testing.T) {
	scriptPath := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	contents, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read migration harness: %v", err)
	}
	script := string(contents)

	for _, required := range []string{
		`HARNESS_READY_TIMEOUT_SECONDS`,
		`MySQL init process done. Ready for start up.`,
		`--protocol=tcp -h127.0.0.1`,
		`authenticated_ready=1`,
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("migration harness missing readiness contract %q", required)
		}
	}
	if strings.Contains(script, `for attempt in $(seq 1 60)`) {
		t.Fatal("migration harness still uses an implicit fixed-attempt readiness loop")
	}
	if strings.Contains(script, `grep -Fq 'ready for connections'`) {
		t.Fatal("migration harness can accept the temporary initialization server marker")
	}
}

func TestCanonicalIdentityAuthoritativeLineageManifest(t *testing.T) {
	const expected = `# source_commit=164788c9c6ce99c825c3587ae314dcb90e62b3ca
# sha256 filename
44322db83275b2f5445a79a5b2b9d0d2671e3674d5f7c704440377dde9146863 028_create_mobile_device_token_table.sql
77e3283ff6476846f1407590742f365ec45c10ad492a8008453a9a8b5fe62461 029_extend_mobile_device_token_invalid_state.sql
13db8bbdf57e05fa5b8cd4faa760ee8420ec0d2d3f416b26a6dabd11d3d346a9 030_create_mobile_refresh_token_table.sql
26d69c94f5a55531d49ad41bbfe0e4e953bc18c871227d9ee402044d5734340c 031_extend_mobile_device_token_apns_metadata.sql
5247277caaf8afbd0cc2053270898eef335e6e75a18b377228453e95887e1969 032_allow_android_push_tokens_without_apns_metadata.sql
355a5c503bbc094a096c564e50393d76fff6e352508aab205bca9ba13322cdbf 033_backfill_android_push_token_metadata_and_length.sql
6732ded58bbf19f03fbccb90c82198304be9d7819d89d3e6a4decdd5ea7c1522 034_create_push_outbox.sql
1bbd046048a322f39cfbc313269528900bd59df1df9320cacb3e6fd10dce4500 035_create_push_preference.sql
f75a52f9f1ee58d2973bc63db33955bf418665e08772ef91f2dfb059e01ba139 036_extend_mobile_refresh_token_for_rotation.sql
9760072a8a05de7a84c73195f74d5577f3efcc15f68fb044ebc909a37162cd6f 037_harden_member_social_links.sql
1509012ae1aae9aae0554081dbd940dba04f4d15b12e591f13c340308ae59660 038_create_auth_principal_tables.sql
56b3e48ca94196f4bf2fa5bb93dad5d7f42df12671a4a2df096d889f6dd742a2 039_create_social_link_continuation.sql
`
	assertFileEquals(t,
		filepath.Join(canonicalMigrationDir(), "testdata", "canonical_identity_authoritative_lineage.sha256"),
		expected,
	)
	assertFileContains(t, filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh"), []string{
		"verify_authoritative_testdata_lineage",
		"AUTHORITATIVE_TESTDATA_LINEAGE=PASS",
		"authoritative manifest cardinality mismatch",
		"authoritative manifest duplicate migration number",
	})
}

func TestCanonicalIdentityCurrentBranchDivergenceIsFrozen(t *testing.T) {
	fixture := filepath.Join(
		"..", "..", "migrations", "testdata", "current-branch-8dcba0b",
		"028_social_auth_security.sql",
	)
	contents, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read frozen current-branch migration: %v", err)
	}
	want := "dc265473bab99621cf06f07e450c64501d1dcbee457308dca528f65933e75513"
	if got := fmt.Sprintf("%x", sha256.Sum256(contents)); got != want {
		t.Fatalf("frozen current-branch migration digest = %s, want %s", got, want)
	}

	harness, err := os.ReadFile(filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh"))
	if err != nil {
		t.Fatalf("read migration harness: %v", err)
	}
	script := string(harness)
	if !strings.Contains(script, `${TESTDATA_DIR}/current-branch-8dcba0b/028_social_auth_security.sql`) {
		t.Fatal("current-branch expected-RED lane is not bound to the frozen fixture")
	}
	if !strings.Contains(script, `if [ "$MODE" = "--expect-current-red" ]; then
  run_current_branch_lineage_probe`) {
		t.Fatal("current-branch expected-RED lane is not isolated from --verify")
	}
}

func TestCanonicalIdentityCandidateManifestIsFailClosed(t *testing.T) {
	harness, err := os.ReadFile(filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh"))
	if err != nil {
		t.Fatalf("read migration harness: %v", err)
	}
	script := string(harness)
	for _, marker := range []string{
		"candidate_manifest_is_valid",
		"candidate manifest duplicate migration number",
		"candidate manifest cardinality mismatch",
		"candidate source migration number cardinality mismatch",
	} {
		if !strings.Contains(script, marker) {
			t.Fatalf("candidate manifest verifier is missing %q", marker)
		}
	}
}

func TestCanonicalIdentityCandidatePacketMatchesCurrentSource(t *testing.T) {
	script := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	command := approvedCurrentCandidateCommand(t, script, "--check-candidate-range")
	output, runErr := command.CombinedOutput()
	if runErr != nil {
		t.Fatalf("current candidate packet failed: %v\n%s", runErr, output)
	}
	if strings.TrimSpace(string(output)) != "CANONICAL_CANDIDATE_RANGE=PASS maintenance_excluded=043" {
		t.Fatalf("unexpected current candidate packet output: %q", output)
	}
}

func TestCanonicalIdentitySchemaContractPinsExactMetadataFingerprints(t *testing.T) {
	contractPath := filepath.Join(canonicalMigrationDir(), "testdata", "canonical_identity_schema_contract.sql")
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read canonical schema contract: %v", err)
	}
	for _, marker := range []string{
		"canonical_schema_columns_fingerprint_v1",
		"canonical_schema_indexes_fingerprint_v1",
		"canonical_schema_foreign_keys_fingerprint_v1",
		"canonical_schema_tables_fingerprint_v1",
		"canonical_schema_triggers_fingerprint_v1",
		"canonical_schema_history_fingerprint_v1",
	} {
		if !bytes.Contains(contract, []byte(marker)) {
			t.Fatalf("canonical schema contract is missing exact metadata marker %q", marker)
		}
	}
}

func approvedCurrentCandidateCommand(t *testing.T, script string, arguments ...string) *exec.Cmd {
	t.Helper()
	testdata := filepath.Join(canonicalMigrationDir(), "testdata")
	candidateManifest, err := os.ReadFile(filepath.Join(testdata, "canonical_identity_candidate_lineage.sha256"))
	if err != nil {
		t.Fatalf("read candidate manifest: %v", err)
	}
	schemaManifest, err := os.ReadFile(filepath.Join(testdata, "canonical_identity_schema_contract.sha256"))
	if err != nil {
		t.Fatalf("read schema contract manifest: %v", err)
	}

	command := exec.Command(script, arguments...)
	command.Env = append(os.Environ(),
		fmt.Sprintf("CANONICAL_CANDIDATE_MANIFEST_SHA256=%x", sha256.Sum256(candidateManifest)),
		fmt.Sprintf("CANONICAL_SCHEMA_CONTRACT_MANIFEST_SHA256=%x", sha256.Sum256(schemaManifest)),
	)
	return command
}

func TestCanonicalIdentityStandaloneFixtureLineage(t *testing.T) {
	script, testdata := copyStandaloneHarnessFixture(t, "mariadb@sha256:e4e5e5e2fb7c089688ddb55cc5ef38c9acff4aeb0aa25375f92f0708795b7a1c\n")
	command := exec.Command(script, "--check-fixture-lineage")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("valid standalone fixture lineage failed: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "HARNESS_FIXTURE_LINEAGE=PASS entries=12" {
		t.Fatalf("unexpected fixture-lineage output: %q", output)
	}

	drifted := filepath.Join(testdata, "authoritative-164788c", "kakao_auth_028_035_fixture.sql")
	if err := os.WriteFile(drifted, []byte("SELECT 'drift';\n"), 0o600); err != nil {
		t.Fatalf("mutate authoritative fixture: %v", err)
	}
	command = exec.Command(script, "--check-fixture-lineage")
	output, err = command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "harness fixture digest mismatch") {
		t.Fatalf("fixture drift was not rejected before Docker: err=%v output=%s", err, output)
	}

	t.Run("duplicate_omits_approved_fixture", func(t *testing.T) {
		script, testdata := copyStandaloneHarnessFixture(t, "mariadb@sha256:e4e5e5e2fb7c089688ddb55cc5ef38c9acff4aeb0aa25375f92f0708795b7a1c\n")
		manifest := filepath.Join(testdata, "canonical_identity_harness_fixtures.sha256")
		contents, readErr := os.ReadFile(manifest)
		if readErr != nil {
			t.Fatalf("read copied fixture manifest: %v", readErr)
		}
		lines := strings.Split(strings.TrimSpace(string(contents)), "\n")
		lines[len(lines)-1] = lines[0]
		writeTestFile(t, manifest, strings.Join(lines, "\n")+"\n")
		output, runErr := exec.Command(script, "--check-fixture-lineage").CombinedOutput()
		if runErr == nil {
			t.Fatalf("duplicate/omitted fixture manifest passed: %s", output)
		}
	})

	t.Run("payload_and_manifest_codrift", func(t *testing.T) {
		script, testdata := copyStandaloneHarnessFixture(t, "mariadb@sha256:e4e5e5e2fb7c089688ddb55cc5ef38c9acff4aeb0aa25375f92f0708795b7a1c\n")
		fixtureName := "authoritative-164788c/kakao_auth_028_035_fixture.sql"
		fixtureContents := "SELECT 'co-drift';\n"
		writeTestFile(t, filepath.Join(testdata, fixtureName), fixtureContents)
		manifest := filepath.Join(testdata, "canonical_identity_harness_fixtures.sha256")
		contents, readErr := os.ReadFile(manifest)
		if readErr != nil {
			t.Fatalf("read copied fixture manifest: %v", readErr)
		}
		lines := strings.Split(strings.TrimSpace(string(contents)), "\n")
		for index, line := range lines {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[1] == fixtureName {
				lines[index] = fmt.Sprintf("%x %s", sha256.Sum256([]byte(fixtureContents)), fixtureName)
			}
		}
		writeTestFile(t, manifest, strings.Join(lines, "\n")+"\n")
		output, runErr := exec.Command(script, "--check-fixture-lineage").CombinedOutput()
		if runErr == nil {
			t.Fatalf("fixture/manifest co-drift passed: %s", output)
		}
	})
}

func TestCanonicalIdentityStandaloneEnginePin(t *testing.T) {
	script, _ := copyStandaloneHarnessFixture(t, "mariadb:latest\n")
	command := exec.Command(script, "--check-fixture-lineage")
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "MariaDB image pin mismatch") {
		t.Fatalf("engine pin drift was not rejected: err=%v output=%s", err, output)
	}
}

func TestCanonicalIdentityStandaloneAuthoritativeManifestRejectsCoDrift(t *testing.T) {
	script, testdata := copyStandaloneHarnessFixture(t, "mariadb@sha256:e4e5e5e2fb7c089688ddb55cc5ef38c9acff4aeb0aa25375f92f0708795b7a1c\n")
	migrations := filepath.Dir(testdata)
	baselineOutput, baselineErr := exec.Command(script, "--check-source-lineage").CombinedOutput()
	if baselineErr != nil {
		t.Fatalf("valid authoritative standalone packet failed: %v\n%s", baselineErr, baselineOutput)
	}
	manifest := filepath.Join(testdata, "canonical_identity_authoritative_lineage.sha256")
	filename := "036_extend_mobile_refresh_token_for_rotation.sql"
	contents := "SELECT 'authoritative-co-drift';\n"
	writeTestFile(t, filepath.Join(migrations, filename), contents)
	manifestBytes, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatalf("read copied authoritative manifest: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(manifestBytes)), "\n")
	for index, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == filename {
			lines[index] = fmt.Sprintf("%x %s", sha256.Sum256([]byte(contents)), filename)
		}
	}
	writeTestFile(t, manifest, strings.Join(lines, "\n")+"\n")
	output, runErr := exec.Command(script, "--check-source-lineage").CombinedOutput()
	if runErr == nil {
		t.Fatalf("authoritative payload/manifest co-drift passed: %s", output)
	}
}

func TestCanonicalIdentityCandidatePacketBehavior(t *testing.T) {
	mutations := map[string]func(t *testing.T, migrations, testdata string){
		"schema_extra_entry": func(t *testing.T, _, testdata string) {
			appendFile(t, filepath.Join(testdata, "canonical_identity_schema_contract.sha256"), strings.Repeat("0", 64)+" unauthorized.sql\n")
		},
		"schema_uppercase_digest": func(t *testing.T, _, testdata string) {
			path := filepath.Join(testdata, "canonical_identity_schema_contract.sha256")
			contents, _ := os.ReadFile(path)
			writeTestFile(t, path, strings.ToUpper(string(contents)))
		},
		"schema_wrong_digest": func(t *testing.T, _, testdata string) {
			writeTestFile(t, filepath.Join(testdata, "canonical_identity_schema_contract.sha256"), strings.Repeat("0", 64)+" canonical_identity_schema_contract.sql\n")
		},
		"schema_malformed_digest": func(t *testing.T, _, testdata string) {
			writeTestFile(t, filepath.Join(testdata, "canonical_identity_schema_contract.sha256"), "abc canonical_identity_schema_contract.sql\n")
		},
		"candidate_duplicate_number": func(t *testing.T, _, testdata string) {
			manifest := filepath.Join(testdata, "canonical_identity_candidate_lineage.sha256")
			contents, _ := os.ReadFile(manifest)
			appendFile(t, manifest, strings.Split(string(contents), "\n")[0]+"\n")
		},
		"candidate_uppercase_digest": func(t *testing.T, _, testdata string) {
			manifest := filepath.Join(testdata, "canonical_identity_candidate_lineage.sha256")
			contents, _ := os.ReadFile(manifest)
			lines := strings.Split(string(contents), "\n")
			fields := strings.Fields(lines[0])
			lines[0] = strings.ToUpper(fields[0]) + " " + fields[1]
			writeTestFile(t, manifest, strings.Join(lines, "\n"))
		},
		"candidate_wrong_digest": func(t *testing.T, _, testdata string) {
			manifest := filepath.Join(testdata, "canonical_identity_candidate_lineage.sha256")
			contents, _ := os.ReadFile(manifest)
			lines := strings.Split(string(contents), "\n")
			fields := strings.Fields(lines[0])
			lines[0] = strings.Repeat("0", 64) + " " + fields[1]
			writeTestFile(t, manifest, strings.Join(lines, "\n"))
		},
		"candidate_malformed_digest": func(t *testing.T, _, testdata string) {
			manifest := filepath.Join(testdata, "canonical_identity_candidate_lineage.sha256")
			contents, _ := os.ReadFile(manifest)
			lines := strings.Split(string(contents), "\n")
			fields := strings.Fields(lines[0])
			lines[0] = "abc " + fields[1]
			writeTestFile(t, manifest, strings.Join(lines, "\n"))
		},
		"candidate_wrong_filename": func(t *testing.T, _, testdata string) {
			manifest := filepath.Join(testdata, "canonical_identity_candidate_lineage.sha256")
			contents, _ := os.ReadFile(manifest)
			writeTestFile(t, manifest, strings.Replace(string(contents), "041_create_canonical_identity_schema.sql", "041_wrong.sql", 1))
		},
		"candidate_missing_source": func(t *testing.T, migrations, _ string) {
			if err := os.Remove(filepath.Join(migrations, "041_create_canonical_identity_schema.sql")); err != nil {
				t.Fatalf("remove candidate source: %v", err)
			}
		},
		"candidate_multiple_sources": func(t *testing.T, migrations, _ string) {
			writeTestFile(t, filepath.Join(migrations, "041_extra.sql"), "SELECT 1;\n")
		},
		"candidate_source_symlink": func(t *testing.T, migrations, _ string) {
			target := filepath.Join(migrations, "041_create_canonical_identity_schema.sql")
			if err := os.Remove(target); err != nil {
				t.Fatalf("remove candidate before symlink: %v", err)
			}
			if err := os.Symlink("040_convert_auth_transaction_boundary_to_innodb.sql", target); err != nil {
				t.Fatalf("symlink candidate: %v", err)
			}
		},
		"candidate_manifest_symlink": func(t *testing.T, _, testdata string) {
			replaceWithSymlink(t, filepath.Join(testdata, "canonical_identity_candidate_lineage.sha256"), "canonical_identity_schema_contract.sha256")
		},
		"schema_contract_symlink": func(t *testing.T, _, testdata string) {
			replaceWithSymlink(t, filepath.Join(testdata, "canonical_identity_schema_contract.sql"), "canonical_identity_runner_fixture.sql")
		},
		"schema_manifest_symlink": func(t *testing.T, _, testdata string) {
			replaceWithSymlink(t, filepath.Join(testdata, "canonical_identity_schema_contract.sha256"), "canonical_identity_candidate_lineage.sha256")
		},
		"schema_marker_only_codrift": func(t *testing.T, _, testdata string) {
			contract := "SELECT 'CANONICAL_SCHEMA_CONTRACT_PASS';\n"
			writeTestFile(t, filepath.Join(testdata, "canonical_identity_schema_contract.sql"), contract)
			writeTestFile(t, filepath.Join(testdata, "canonical_identity_schema_contract.sha256"), fmt.Sprintf("%x canonical_identity_schema_contract.sql\n", sha256.Sum256([]byte(contract))))
		},
		"candidate_payload_manifest_codrift": func(t *testing.T, migrations, testdata string) {
			filename := "040_convert_auth_transaction_boundary_to_innodb.sql"
			contents := "SELECT 400;\n"
			writeTestFile(t, filepath.Join(migrations, filename), contents)
			manifest := filepath.Join(testdata, "canonical_identity_candidate_lineage.sha256")
			manifestBytes, readErr := os.ReadFile(manifest)
			if readErr != nil {
				t.Fatalf("read candidate manifest: %v", readErr)
			}
			lines := strings.Split(strings.TrimSpace(string(manifestBytes)), "\n")
			for index, line := range lines {
				fields := strings.Fields(line)
				if len(fields) == 2 && fields[1] == filename {
					lines[index] = fmt.Sprintf("%x %s", sha256.Sum256([]byte(contents)), filename)
				}
			}
			writeTestFile(t, manifest, strings.Join(lines, "\n")+"\n")
		},
	}

	script, _, _ := copyCandidatePacket(t)
	command := approvedCandidateCommand(t, script, "--check-candidate-range")
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "CANONICAL_CANDIDATE_RANGE=PASS") {
		t.Fatalf("valid future-shaped packet was not accepted: err=%v output=%s", err, output)
	}

	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			script, migrations, testdata := copyCandidatePacket(t)
			command := approvedCandidateCommand(t, script, "--check-candidate-range")
			mutate(t, migrations, testdata)
			output, _ := command.CombinedOutput()
			if strings.Contains(string(output), "CANONICAL_CANDIDATE_RANGE=PASS") {
				t.Fatalf("invalid candidate packet passed: %s", output)
			}
		})
	}
}

func copyCandidatePacket(t *testing.T) (string, string, string) {
	t.Helper()
	script, testdata := copyStandaloneHarnessFixture(t, "mariadb@sha256:e4e5e5e2fb7c089688ddb55cc5ef38c9acff4aeb0aa25375f92f0708795b7a1c\n")
	migrations := filepath.Dir(testdata)
	candidates := map[string]string{
		"040_convert_auth_transaction_boundary_to_innodb.sql": "SELECT 40;\n",
		"041_create_canonical_identity_schema.sql":            "SELECT 41;\n",
		"042_prepare_canonical_auth_cutover.sql":              "SELECT 42;\n",
	}
	var manifest strings.Builder
	for _, filename := range []string{
		"040_convert_auth_transaction_boundary_to_innodb.sql",
		"041_create_canonical_identity_schema.sql",
		"042_prepare_canonical_auth_cutover.sql",
	} {
		contents := candidates[filename]
		writeTestFile(t, filepath.Join(migrations, filename), contents)
		manifest.WriteString(fmt.Sprintf("%x %s\n", sha256.Sum256([]byte(contents)), filename))
	}
	writeTestFile(t, filepath.Join(testdata, "canonical_identity_candidate_lineage.sha256"), manifest.String())
	contract := `-- canonical_identity_schema_contract_v1
SET @canonical_contract_failures =
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='AUTH_IDENTITY' AND COLUMN_NAME='provider') +
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='AUTH_IDENTITY' AND INDEX_NAME='uq_auth_identity_provider_subject') +
  (SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS WHERE CONSTRAINT_SCHEMA=DATABASE() AND TABLE_NAME='AUTH_IDENTITY' AND CONSTRAINT_TYPE='FOREIGN KEY') +
  (SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND EVENT_OBJECT_TABLE='AUTH_IDENTITY') +
  (SELECT COUNT(*) FROM _migration_history WHERE filename IN ('040_convert_auth_transaction_boundary_to_innodb.sql','041_create_canonical_identity_schema.sql','042_prepare_canonical_auth_cutover.sql'));
SELECT IF(@canonical_contract_failures >= 5, 'CANONICAL_SCHEMA_CONTRACT_PASS', 'CANONICAL_SCHEMA_CONTRACT_FAIL');
`
	writeTestFile(t, filepath.Join(testdata, "canonical_identity_schema_contract.sql"), contract)
	writeTestFile(t, filepath.Join(testdata, "canonical_identity_schema_contract.sha256"), fmt.Sprintf("%x canonical_identity_schema_contract.sql\n", sha256.Sum256([]byte(contract))))
	return script, migrations, testdata
}

func approvedCandidateCommand(t *testing.T, script string, arguments ...string) *exec.Cmd {
	t.Helper()
	testdata := filepath.Join(filepath.Dir(filepath.Dir(script)), "migrations", "testdata")
	candidateManifest, err := os.ReadFile(filepath.Join(testdata, "canonical_identity_candidate_lineage.sha256"))
	if err != nil {
		t.Fatalf("read candidate manifest approval input: %v", err)
	}
	schemaManifest, err := os.ReadFile(filepath.Join(testdata, "canonical_identity_schema_contract.sha256"))
	if err != nil {
		t.Fatalf("read schema manifest approval input: %v", err)
	}
	command := exec.Command(script, arguments...)
	command.Env = append(os.Environ(),
		fmt.Sprintf("CANONICAL_CANDIDATE_MANIFEST_SHA256=%x", sha256.Sum256(candidateManifest)),
		fmt.Sprintf("CANONICAL_SCHEMA_CONTRACT_MANIFEST_SHA256=%x", sha256.Sum256(schemaManifest)),
	)
	return command
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func appendFile(t *testing.T, path, contents string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s for append: %v", path, err)
	}
	defer file.Close()
	if _, err := file.WriteString(contents); err != nil {
		t.Fatalf("append %s: %v", path, err)
	}
}

func replaceWithSymlink(t *testing.T, path, target string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove %s before symlink: %v", path, err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("symlink %s: %v", path, err)
	}
}

func copyStandaloneHarnessFixture(t *testing.T, image string) (string, string) {
	t.Helper()
	root := t.TempDir()
	scriptDir := filepath.Join(root, "backend", "scripts")
	testdata := filepath.Join(root, "backend", "migrations", "testdata")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("create harness script directory: %v", err)
	}
	if err := os.MkdirAll(testdata, 0o755); err != nil {
		t.Fatalf("create harness testdata directory: %v", err)
	}

	sourceScript := filepath.Join("..", "..", "scripts", "test-canonical-identity-migrations.sh")
	scriptBytes, err := os.ReadFile(sourceScript)
	if err != nil {
		t.Fatalf("read harness script: %v", err)
	}
	script := filepath.Join(scriptDir, "test-canonical-identity-migrations.sh")
	if err := os.WriteFile(script, scriptBytes, 0o755); err != nil {
		t.Fatalf("write harness script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdata, "mariadb-10.1.38.image"), []byte(image), 0o600); err != nil {
		t.Fatalf("write image pin: %v", err)
	}

	sourceTestdata := filepath.Join(canonicalMigrationDir(), "testdata")
	manifestBytes, err := os.ReadFile(filepath.Join(sourceTestdata, "canonical_identity_harness_fixtures.sha256"))
	if err != nil {
		t.Fatalf("read fixture manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdata, "canonical_identity_harness_fixtures.sha256"), manifestBytes, 0o600); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(manifestBytes)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			t.Fatalf("invalid source fixture manifest line: %q", line)
		}
		source := filepath.Join(sourceTestdata, fields[1])
		destination := filepath.Join(testdata, fields[1])
		contents, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("read fixture %s: %v", fields[1], err)
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatalf("create fixture parent: %v", err)
		}
		if err := os.WriteFile(destination, contents, 0o600); err != nil {
			t.Fatalf("write fixture %s: %v", fields[1], err)
		}
	}

	authoritativeManifest, err := os.ReadFile(filepath.Join(sourceTestdata, "canonical_identity_authoritative_lineage.sha256"))
	if err != nil {
		t.Fatalf("read authoritative manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdata, "canonical_identity_authoritative_lineage.sha256"), authoritativeManifest, 0o600); err != nil {
		t.Fatalf("write authoritative manifest: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(authoritativeManifest)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		filename := fields[1]
		number := strings.SplitN(filename, "_", 2)[0]
		source := filepath.Join(canonicalMigrationDir(), filename)
		if number >= "028" && number <= "035" {
			source = filepath.Join(sourceTestdata, "authoritative-164788c", filename)
		}
		contents, readErr := os.ReadFile(source)
		if readErr != nil {
			t.Fatalf("read authoritative migration %s: %v", filename, readErr)
		}
		if writeErr := os.WriteFile(filepath.Join(filepath.Dir(testdata), filename), contents, 0o600); writeErr != nil {
			t.Fatalf("write authoritative migration %s: %v", filename, writeErr)
		}
		if number >= "028" && number <= "035" {
			destination := filepath.Join(testdata, "authoritative-164788c", filename)
			if mkdirErr := os.MkdirAll(filepath.Dir(destination), 0o755); mkdirErr != nil {
				t.Fatalf("create authoritative predecessor directory: %v", mkdirErr)
			}
			if writeErr := os.WriteFile(destination, contents, 0o600); writeErr != nil {
				t.Fatalf("write authoritative predecessor %s: %v", filename, writeErr)
			}
		}
	}
	return script, testdata
}

func assertFileEquals(t *testing.T, path, want string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(contents) != want {
		t.Fatalf("%s does not match the pinned content", path)
	}
}

func assertFileContains(t *testing.T, path string, markers []string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, marker := range markers {
		if !strings.Contains(string(contents), marker) {
			t.Errorf("%s is missing %q", path, marker)
		}
	}
}
