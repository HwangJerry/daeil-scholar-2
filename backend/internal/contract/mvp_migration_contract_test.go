// mvp_migration_contract_test.go — Validates MVP schema migrations and MariaDB 10.1 compatibility.
package contract

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type migrationContract struct {
	name     string
	required []string
}

func TestMVPMigrationSetContainsRequiredSchema(t *testing.T) {
	for _, contract := range mvpMigrationContracts() {
		t.Run(contract.name, func(t *testing.T) {
			sql := readMigrationFile(t, contract.name)
			for _, marker := range contract.required {
				if !strings.Contains(sql, marker) {
					t.Errorf("%s is missing %q", contract.name, marker)
				}
			}
		})
	}
}

func TestMVPMigrationsAvoidUnsupportedMariaDB101Syntax(t *testing.T) {
	forbidden := map[string]*regexp.Regexp{
		"common table expression": regexp.MustCompile(`(?i)\bWITH\s+[A-Z0-9_]+\s+AS\s*\(`),
		"window function":         regexp.MustCompile(`(?i)\b(ROW_NUMBER|DENSE_RANK|RANK)\s*\(`),
		"JSON column":             regexp.MustCompile(`(?i)\bJSON\b`),
		"default expression":      regexp.MustCompile(`(?i)\bDEFAULT\s*\(`),
		"datetime default":        regexp.MustCompile(`(?i)\bDEFAULT\s+CURRENT_TIMESTAMP\b`),
		"check constraint":        regexp.MustCompile(`(?i)\bCHECK\s*\(`),
	}

	for _, name := range mvpMigrationNames() {
		t.Run(name, func(t *testing.T) {
			sql := stripSQLComments(readMigrationFile(t, name))
			for feature, pattern := range forbidden {
				if pattern.MatchString(sql) {
					t.Errorf("%s uses unsupported %s syntax", name, feature)
				}
			}
		})
	}
}

func TestApplyAllIncludesMigrationsBeforeHelperCleanup(t *testing.T) {
	sql := readMigrationFile(t, "apply_all.sql")
	cleanup := strings.Index(sql, "Cleanup: Drop helper procedures")
	if cleanup < 0 {
		t.Fatal("apply_all.sql is missing helper cleanup")
	}
	for number := 30; number <= 34; number++ {
		marker := fmt.Sprintf("-- %03d:", number)
		position := strings.Index(sql, marker)
		if position < 0 {
			t.Errorf("apply_all.sql is missing section %s", marker)
		} else if position > cleanup {
			t.Errorf("apply_all.sql section %s appears after helper cleanup", marker)
		}
	}
}

func TestPushPreferenceMigrationExtendsPreexistingSchema(t *testing.T) {
	standalone := readMigrationFile(t, "032_create_push_device_and_preferences.sql")
	applyAll := readMigrationFile(t, "apply_all.sql")
	runbook := readMigrationFile(t, "MVP_ROLLBACK.md")

	for name, content := range map[string]string{
		"standalone": standalone,
		"apply_all":  applyAll,
	} {
		addPreview := regexp.MustCompile(`(?s)CALL\s+_add_column_if_not_exists\s*\(\s*'ALUMNI_PUSH_PREFERENCE'\s*,\s*'MESSAGE_PREVIEW_ENABLED'`)
		if !addPreview.MatchString(content) {
			t.Fatalf("%s must add MESSAGE_PREVIEW_ENABLED when ALUMNI_PUSH_PREFERENCE already exists", name)
		}
	}
	for _, marker := range []string{"pre-existing `ALUMNI_PUSH_PREFERENCE`", "preserve the table", "MESSAGE_PREVIEW_ENABLED"} {
		if !strings.Contains(runbook, marker) {
			t.Fatalf("rollback runbook missing pre-existing push preference guidance %q", marker)
		}
	}
}

func TestApplyAllIncludesEveryNumberedMigration(t *testing.T) {
	sql := readMigrationFile(t, "apply_all.sql")
	for number := 1; number <= 34; number++ {
		marker := fmt.Sprintf("-- %03d:", number)
		if !strings.Contains(sql, marker) {
			t.Errorf("apply_all.sql is missing section %s", marker)
		}
	}
}

func TestApplyAllAvoidsANSIQuotesSensitiveHelperArguments(t *testing.T) {
	sql := stripSQLComments(readMigrationFile(t, "apply_all.sql"))
	unsafe := regexp.MustCompile(`(?m)CALL\s+_add_[^(]+\([^\n]*,\s*"`)
	if unsafe.MatchString(sql) {
		t.Error("apply_all.sql passes double-quoted SQL definitions to helpers")
	}
}

func TestApplyAllGuardsOneTimePrivacyBackfill(t *testing.T) {
	sql := readMigrationFile(t, "apply_all.sql")
	for _, marker := range []string{
		"ALUMNI_SCHEMA_MIGRATION",
		"@run_021",
		"WHERE @run_021 = 1",
		"'021_default_privacy_private'",
	} {
		if !strings.Contains(sql, marker) {
			t.Errorf("apply_all.sql is missing one-time privacy guard %q", marker)
		}
	}
}

func TestApplyAllIndexHelpersValidateExistingDefinitions(t *testing.T) {
	sql := readMigrationFile(t, "apply_all.sql")
	for _, marker := range []string{"GROUP_CONCAT", "SEQ_IN_INDEX", "NON_UNIQUE", "@idx_columns"} {
		if !strings.Contains(sql, marker) {
			t.Errorf("apply_all.sql index helpers do not validate existing definitions: missing %q", marker)
		}
	}
}

func TestApplyAllMirrorsRequiredMVPSchema(t *testing.T) {
	sql := readMigrationFile(t, "apply_all.sql")
	start := strings.Index(sql, "-- 030:")
	end := strings.Index(sql, "Cleanup: Drop helper procedures")
	if start < 0 || end < 0 || start >= end {
		t.Fatal("apply_all.sql does not contain a valid 030-034 section")
	}
	segment := sql[start:end]
	for _, contract := range mvpMigrationContracts() {
		for _, marker := range contract.required {
			if !strings.Contains(segment, marker) {
				t.Errorf("apply_all.sql is missing %q from %s", marker, contract.name)
			}
		}
	}
}

func TestMVPMigrationsHaveRollbackRunbookAndNoPublicDonorSchema(t *testing.T) {
	runbook := readMigrationFile(t, "MVP_ROLLBACK.md")
	for _, marker := range []string{"034", "033", "032", "031", "030", "backup", "information_schema"} {
		if !strings.Contains(strings.ToLower(runbook), strings.ToLower(marker)) {
			t.Errorf("rollback runbook is missing %q", marker)
		}
	}

	for _, name := range mvpMigrationNames() {
		sql := strings.ToUpper(stripSQLComments(readMigrationFile(t, name)))
		for _, forbidden := range []string{"PUBLIC_DONOR", "DONOR_LIST", "DONOR_DIRECTORY"} {
			if strings.Contains(sql, forbidden) {
				t.Errorf("%s creates forbidden public donor schema %q", name, forbidden)
			}
		}
	}
}

func TestLegacyRejectedMembersAreBackfilledAsRejected(t *testing.T) {
	for _, name := range []string{"030_create_admin_role_and_alumni_verification.sql", "apply_all.sql"} {
		sql := readMigrationFile(t, name)
		for _, marker := range []string{"USR_STATUS = 'BAA'", "THEN 'rejected'"} {
			if !strings.Contains(sql, marker) {
				t.Errorf("%s does not preserve legacy rejected members: missing %q", name, marker)
			}
		}
	}
}

func TestDonationBackfillFailsClosedOnAmbiguousLegacyRows(t *testing.T) {
	for _, name := range []string{"033_extend_weo_order_for_donation_ledger.sql", "apply_all.sql"} {
		sql := readMigrationFile(t, name)
		for _, marker := range []string{
			"_MVP_DONATION_PREFLIGHT_GUARD",
			"O_STATUS = 'N'",
			"O_PAYMENT = 'Y' AND (O_STATUS IS NULL OR O_STATUS <> 'Y')",
			"O_STATUS = 'Y' AND (O_PAYMENT IS NULL OR O_PAYMENT <> 'Y')",
			"O_PRICE < 0",
			"O_PAY < 0",
			"O_PRICE IS NULL",
			"COALESCE(O_STATUS, '') NOT IN ('I','Y','N')",
			"COALESCE(O_PAYMENT, '') NOT IN ('Y','N')",
			"O_PRICE <> O_PAY",
			"COALESCE(O_PAYDATE, O_REGDATE) IS NULL",
			"O_PAYMENT = 'Y' AND O_STATUS = 'Y'",
			"THEN O_PAY",
			"ELSE 0",
		} {
			if !strings.Contains(sql, marker) {
				t.Errorf("%s is missing donation preflight/backfill marker %q", name, marker)
			}
		}
		if strings.Contains(sql, "O_PAYMENT = 'Y' OR O_STATUS = 'Y'") {
			t.Errorf("%s treats inconsistent payment state as completed", name)
		}
	}
}

func mvpMigrationNames() []string {
	contracts := mvpMigrationContracts()
	names := make([]string, 0, len(contracts))
	for _, contract := range contracts {
		names = append(names, contract.name)
	}
	return names
}

func mvpMigrationContracts() []migrationContract {
	return []migrationContract{
		{
			name: "030_create_admin_role_and_alumni_verification.sql",
			required: []string{
				"ALUMNI_ADMIN_ROLE", "ALUMNI_VERIFICATION", "unsubmitted",
				"reapproval_pending", "REJECTION_REASON", "REVIEWED_BY",
			},
		},
		{
			name: "031_create_member_block_and_message_retention.sql",
			required: []string{
				"ALUMNI_MEMBER_BLOCK", "BLOCKER_USR_SEQ", "BLOCKED_USR_SEQ",
				"AM_CLIENT_MESSAGE_ID", "AM_VISIBLE_RECVR", "PURGE_AT",
			},
		},
		{
			name: "032_create_push_device_and_preferences.sql",
			required: []string{
				"ALUMNI_PUSH_DEVICE", "DEVICE_TOKEN", "APNS_ENVIRONMENT",
				"ALUMNI_PUSH_PREFERENCE", "MESSAGE_PREVIEW_ENABLED",
			},
		},
		{
			name: "033_extend_weo_order_for_donation_ledger.sql",
			required: []string{
				"WEO_ORDER", "O_SOURCE", "O_TRANSACTION_NO", "O_COMPOSITE_KEY",
				"O_GROSS_AMOUNT", "O_REFUNDED_AMOUNT", "O_NET_RECEIVED_AMOUNT",
				"O_LIFECYCLE_STATUS", "O_DONATION_DATE",
			},
		},
		{
			name: "034_add_personal_data_retention_support.sql",
			required: []string{
				"USR_ANONYMIZED_AT", "USR_PURGE_AT", "O_LEGAL_RETENTION_UNTIL",
				"O_ACCOUNT_UNLINKED_AT", "AM_SENDER_ANONYMIZED_YN",
				"O_ACCOUNT_USR_SEQ", "AM_SENDER_ACCOUNT_SEQ", "AM_RECVR_ACCOUNT_SEQ",
			},
		},
	}
}

func readMigrationFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(canonicalMigrationDir(), name))
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	return string(data)
}

func canonicalMigrationDir() string {
	return filepath.Join("..", "..", "migrations")
}

func stripSQLComments(sql string) string {
	lines := strings.Split(sql, "\n")
	kept := lines[:0]
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") || trimmed == "" {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
