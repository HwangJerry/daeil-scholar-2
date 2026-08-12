// canonical_identity_migration_provenance_test.go — Pins immutable canonical-auth migration bytes.
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

func TestCanonicalIdentityMigrationProvenance(t *testing.T) {
	expected := map[string]string{
		"036_extend_mobile_refresh_token_for_rotation.sql": "f75a52f9f1ee58d2973bc63db33955bf418665e08772ef91f2dfb059e01ba139",
		"037_harden_member_social_links.sql":               "9760072a8a05de7a84c73195f74d5577f3efcc15f68fb044ebc909a37162cd6f",
		"038_create_auth_principal_tables.sql":             "1509012ae1aae9aae0554081dbd940dba04f4d15b12e591f13c340308ae59660",
		"039_create_social_link_continuation.sql":          "56b3e48ca94196f4bf2fa5bb93dad5d7f42df12671a4a2df096d889f6dd742a2",
	}

	for name, want := range expected {
		t.Run(name, func(t *testing.T) {
			contents, err := os.ReadFile(filepath.Join(canonicalMigrationDir(), name))
			if err != nil {
				t.Fatalf("read immutable migration: %v", err)
			}
			got := fmt.Sprintf("%x", sha256.Sum256(contents))
			if got != want {
				t.Fatalf("immutable migration digest = %s, want %s", got, want)
			}
		})
	}
}

func TestCanonicalIdentityActiveProductionLineageMatchesAuthoritative(t *testing.T) {
	expected := map[string]string{
		"028_create_mobile_device_token_table.sql":                "44322db83275b2f5445a79a5b2b9d0d2671e3674d5f7c704440377dde9146863",
		"029_extend_mobile_device_token_invalid_state.sql":        "77e3283ff6476846f1407590742f365ec45c10ad492a8008453a9a8b5fe62461",
		"030_create_mobile_refresh_token_table.sql":               "13db8bbdf57e05fa5b8cd4faa760ee8420ec0d2d3f416b26a6dabd11d3d346a9",
		"031_extend_mobile_device_token_apns_metadata.sql":        "26d69c94f5a55531d49ad41bbfe0e4e953bc18c871227d9ee402044d5734340c",
		"032_allow_android_push_tokens_without_apns_metadata.sql": "5247277caaf8afbd0cc2053270898eef335e6e75a18b377228453e95887e1969",
		"033_backfill_android_push_token_metadata_and_length.sql": "355a5c503bbc094a096c564e50393d76fff6e352508aab205bca9ba13322cdbf",
		"034_create_push_outbox.sql":                              "6732ded58bbf19f03fbccb90c82198304be9d7819d89d3e6a4decdd5ea7c1522",
		"035_create_push_preference.sql":                          "1bbd046048a322f39cfbc313269528900bd59df1df9320cacb3e6fd10dce4500",
	}

	for name, want := range expected {
		t.Run(name, func(t *testing.T) {
			contents, err := os.ReadFile(filepath.Join(canonicalMigrationDir(), name))
			if err != nil {
				t.Fatalf("read active authoritative migration: %v", err)
			}
			got := fmt.Sprintf("%x", sha256.Sum256(contents))
			if got != want {
				t.Fatalf("active authoritative migration digest = %s, want %s", got, want)
			}
		})
	}
}

func TestCanonicalIdentityAuthoritativePredecessorProvenance(t *testing.T) {
	expected := map[string]string{
		"028_create_mobile_device_token_table.sql":                "44322db83275b2f5445a79a5b2b9d0d2671e3674d5f7c704440377dde9146863",
		"029_extend_mobile_device_token_invalid_state.sql":        "77e3283ff6476846f1407590742f365ec45c10ad492a8008453a9a8b5fe62461",
		"030_create_mobile_refresh_token_table.sql":               "13db8bbdf57e05fa5b8cd4faa760ee8420ec0d2d3f416b26a6dabd11d3d346a9",
		"031_extend_mobile_device_token_apns_metadata.sql":        "26d69c94f5a55531d49ad41bbfe0e4e953bc18c871227d9ee402044d5734340c",
		"032_allow_android_push_tokens_without_apns_metadata.sql": "5247277caaf8afbd0cc2053270898eef335e6e75a18b377228453e95887e1969",
		"033_backfill_android_push_token_metadata_and_length.sql": "355a5c503bbc094a096c564e50393d76fff6e352508aab205bca9ba13322cdbf",
		"034_create_push_outbox.sql":                              "6732ded58bbf19f03fbccb90c82198304be9d7819d89d3e6a4decdd5ea7c1522",
		"035_create_push_preference.sql":                          "1bbd046048a322f39cfbc313269528900bd59df1df9320cacb3e6fd10dce4500",
		"kakao_auth_028_035_fixture.sql":                          "c7bc3d29ca7690a43b284937a466da2b70f80f97a44b3bc4ca683e3aa694e397",
		"kakao_auth_edge_cases.sql":                               "4c90fb6c0852080adc96db0393ae6ba522b304e9ad44383449ed31dbbd9ea0c5",
	}

	root := filepath.Join(canonicalMigrationDir(), "testdata", "authoritative-164788c")
	for name, want := range expected {
		t.Run(name, func(t *testing.T) {
			contents, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatalf("read authoritative predecessor: %v", err)
			}
			got := fmt.Sprintf("%x", sha256.Sum256(contents))
			if got != want {
				t.Fatalf("authoritative predecessor digest = %s, want %s", got, want)
			}
		})
	}
}

func TestMigrationRunnerDiscoversOnlyNumberedMigrations(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "migrate.sh"))
	if err != nil {
		t.Fatalf("read migration runner: %v", err)
	}

	script := string(contents)
	if strings.Contains(script, `"$MIGRATIONS_DIR"/*.sql`) {
		t.Fatal("migration runner discovers unnumbered SQL such as apply_all.sql")
	}
	const numberedGlob = `"$MIGRATIONS_DIR"/[0-9][0-9][0-9]_*.sql`
	if count := strings.Count(script, numberedGlob); count != 5 {
		t.Fatalf("numbered migration glob appears %d times, want source, approval-count, approval-closure, seed, and apply loops", count)
	}
}

func TestMigrationRunnerKeepsCredentialsOutOfProcessArguments(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "migrate.sh"))
	if err != nil {
		t.Fatalf("read migration runner: %v", err)
	}
	script := string(contents)

	for _, forbidden := range []string{"MYSQL_CMD=", `$MYSQL_CMD`, `-p${DB_PASSWORD}`, "xargs"} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("migration runner contains unsafe command construction %q", forbidden)
		}
	}
	for _, required := range []string{
		`MYSQL_ARGS=(--defaults-extra-file="$MYSQL_OPTION_FILE")`,
		`mysql "${MYSQL_ARGS[@]}"`,
		`chmod 600 "$MYSQL_OPTION_FILE"`,
		`trap cleanup EXIT`,
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("migration runner missing option-file contract %q", required)
		}
	}
}

func TestMigrationRunnerValidatesFilenameAndContentDigest(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "migrate.sh"))
	if err != nil {
		t.Fatalf("read migration runner: %v", err)
	}
	script := string(contents)

	for _, required := range []string{
		`^([0-9]{3})_[a-z0-9][a-z0-9_]*\.sql$`,
		`sha256 CHAR(64) CHARACTER SET ascii NOT NULL`,
		`stored_sha256`,
		`migration content digest mismatch`,
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("migration runner missing filename/digest contract %q", required)
		}
	}
}

func TestMigrationRunnerRequiresExactHistoryAuthority(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "migrate.sh"))
	if err != nil {
		t.Fatalf("read migration runner: %v", err)
	}
	script := string(contents)

	for _, required := range []string{
		`LEFT(filename, 4) = '${number}_'`,
		`LEFT(filename, 4) = '${file_num}_'`,
		`DATA_TYPE='char'`,
		`CHARACTER_SET_NAME='ascii'`,
		`IS_NULLABLE='NO'`,
		`ENGINE='InnoDB'`,
		`invalid_history_digest_count`,
		`history_row_count`,
		`preflight_migration_source_set`,
		`duplicate migration number in source set`,
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("migration runner missing exact history authority contract %q", required)
		}
	}
	if strings.Contains(script, `filename LIKE '${number}\\_%'`) ||
		strings.Contains(script, `filename LIKE '${file_num}\\_%'`) {
		t.Fatal("migration-number collision check remains sensitive to NO_BACKSLASH_ESCAPES")
	}
}

func TestMigrationRunnerUsesOptionFileAtRuntime(t *testing.T) {
	root := t.TempDir()
	migrations := filepath.Join(root, "backend", "migrations")
	fakeBin := filepath.Join(root, "bin")
	if err := os.MkdirAll(migrations, 0o755); err != nil {
		t.Fatalf("create migration fixture directory: %v", err)
	}
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("create fake binary directory: %v", err)
	}

	runner, err := os.ReadFile(filepath.Join("..", "..", "..", "migrate.sh"))
	if err != nil {
		t.Fatalf("read migration runner: %v", err)
	}
	runnerPath := filepath.Join(root, "migrate.sh")
	if err := os.WriteFile(runnerPath, runner, 0o755); err != nil {
		t.Fatalf("write migration runner fixture: %v", err)
	}
	const syntheticPassword = "fixture-password-not-a-secret"
	envContents := "DB_HOST=fixture-db.invalid\nDB_PORT=3306\nDB_USER=fixture-user\nDB_PASSWORD=" + syntheticPassword + "\nDB_NAME=fixture-db\n"
	if err := os.WriteFile(filepath.Join(root, "backend", ".env"), []byte(envContents), 0o600); err != nil {
		t.Fatalf("write synthetic env fixture: %v", err)
	}
	migrationPath := filepath.Join(migrations, "001_fixture.sql")
	if err := os.WriteFile(migrationPath, []byte("SELECT 1;\n"), 0o600); err != nil {
		t.Fatalf("write migration fixture: %v", err)
	}

	firstArgumentLog := filepath.Join(root, "mysql-first-argument.log")
	allArgumentsLog := filepath.Join(root, "mysql-all-arguments.log")
	xargsArgumentsLog := filepath.Join(root, "xargs-arguments.log")
	echoArgumentsLog := filepath.Join(root, "echo-arguments.log")
	fakeMySQL := `#!/bin/sh
printf '%s\n' "$1" >> "$MYSQL_FIRST_ARGUMENT_LOG"
printf '%s\0' "$@" >> "$MYSQL_ALL_ARGUMENTS_LOG"
case "$*" in
  *"DELETE FROM _migration_runner_lock"*) [ "${MYSQL_FAIL_UNLOCK:-0}" = "1" ] && exit 7 ;;
  *"SELECT CONCAT("*"TABLE_NAME='_migration_journal'"*) printf '1:5:5:1:1:1\n' ;;
  *"SELECT CONCAT("*"TABLE_NAME='_migration_runner_lock'"*) printf '1:2:2:1:1:1\n' ;;
  *"SELECT CONCAT("*) printf '1:3:3:1:1:1\n' ;;
  *"sha256 IS NULL"*) printf '0\n' ;;
  *"SELECT"*"FROM _migration_journal j LEFT JOIN _migration_history"*) printf '0\n' ;;
  *"LEFT(filename, 4)"*) printf '0\n' ;;
  *"SELECT COUNT(*) FROM _migration_history WHERE filename="*) printf '0\n' ;;
  *"SELECT COUNT(*) FROM _migration_journal WHERE filename="*) printf '0\n' ;;
  *"SELECT COUNT(*) FROM _migration_journal j JOIN _migration_history"*) printf '1\n' ;;
esac
exit 0
`
	fakeMySQLPath := filepath.Join(fakeBin, "mysql")
	if err := os.WriteFile(fakeMySQLPath, []byte(fakeMySQL), 0o755); err != nil {
		t.Fatalf("write fake mysql: %v", err)
	}
	fakeXargs := `#!/bin/sh
printf '%s\0' "$@" >> "$XARGS_ARGUMENTS_LOG"
exec /usr/bin/xargs "$@"
`
	if err := os.WriteFile(filepath.Join(fakeBin, "xargs"), []byte(fakeXargs), 0o755); err != nil {
		t.Fatalf("write fake xargs: %v", err)
	}
	fakeEcho := `#!/bin/sh
printf '%s\0' "$@" >> "$ECHO_ARGUMENTS_LOG"
exec /bin/echo "$@"
`
	if err := os.WriteFile(filepath.Join(fakeBin, "echo"), []byte(fakeEcho), 0o755); err != nil {
		t.Fatalf("write fake echo: %v", err)
	}

	command := exec.Command(runnerPath, migrationPath)
	command.Env = append(os.Environ(),
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"MYSQL_FIRST_ARGUMENT_LOG="+firstArgumentLog,
		"MYSQL_ALL_ARGUMENTS_LOG="+allArgumentsLog,
		"XARGS_ARGUMENTS_LOG="+xargsArgumentsLog,
		"ECHO_ARGUMENTS_LOG="+echoArgumentsLog,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("migration runner failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "OK    001_fixture.sql") {
		t.Fatalf("migration runner did not apply fixture\n%s", output)
	}

	allArguments, err := os.ReadFile(allArgumentsLog)
	if err != nil {
		t.Fatalf("read fake mysql all-arguments log: %v", err)
	}
	if strings.Contains(string(allArguments), syntheticPassword) {
		t.Fatal("database password leaked into mysql process arguments")
	}
	xargsArguments, err := os.ReadFile(xargsArgumentsLog)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read xargs arguments log: %v", err)
	}
	if strings.Contains(string(xargsArguments), syntheticPassword) {
		t.Fatal("database password leaked into an env-parsing child process argv")
	}
	echoArguments, err := os.ReadFile(echoArgumentsLog)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read echo arguments log: %v", err)
	}
	if strings.Contains(string(echoArguments), syntheticPassword) {
		t.Fatal("database password leaked into a trim-command child process argv")
	}
	firstArguments, err := os.ReadFile(firstArgumentLog)
	if err != nil {
		t.Fatalf("read fake mysql first-argument log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(firstArguments)), "\n")
	if len(lines) == 0 {
		t.Fatal("fake mysql received no calls")
	}
	var optionFile string
	for _, line := range lines {
		firstArgument := strings.Fields(line)[0]
		const prefix = "--defaults-extra-file="
		if !strings.HasPrefix(firstArgument, prefix) {
			t.Fatalf("mysql first argument = %q, want defaults-extra-file", firstArgument)
		}
		if optionFile == "" {
			optionFile = strings.TrimPrefix(firstArgument, prefix)
		}
	}
	if _, err := os.Stat(optionFile); !os.IsNotExist(err) {
		t.Fatalf("temporary mysql option file still exists: %v", err)
	}

	command = exec.Command(runnerPath, migrationPath)
	command.Env = append(os.Environ(),
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"MYSQL_FIRST_ARGUMENT_LOG="+firstArgumentLog,
		"MYSQL_ALL_ARGUMENTS_LOG="+allArgumentsLog,
		"XARGS_ARGUMENTS_LOG="+xargsArgumentsLog,
		"ECHO_ARGUMENTS_LOG="+echoArgumentsLog,
		"MYSQL_FAIL_UNLOCK=1",
	)
	if output, err = command.CombinedOutput(); err == nil {
		t.Fatalf("migration runner unexpectedly passed unlock failure: %s", output)
	}
	firstArguments, err = os.ReadFile(firstArgumentLog)
	if err != nil {
		t.Fatalf("read fake mysql first-argument log after unlock failure: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(firstArguments)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		path := strings.TrimPrefix(fields[0], "--defaults-extra-file=")
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("temporary mysql option file survived unlock failure: %s (%v)", path, statErr)
		}
	}
}
