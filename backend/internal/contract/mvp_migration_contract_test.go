// mvp_migration_contract_test.go — Validates MVP schema migrations and MariaDB 10.1 compatibility.
package contract

import (
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
	// MariaDB 10.1 supports CURRENT_TIMESTAMP defaults for DATETIME columns.
	forbidden := map[string]*regexp.Regexp{
		"common table expression": regexp.MustCompile(`(?i)\bWITH\s+[A-Z0-9_]+\s+AS\s*\(`),
		"window function":         regexp.MustCompile(`(?i)\b(ROW_NUMBER|DENSE_RANK|RANK)\s*\(`),
		"JSON column":             regexp.MustCompile(`(?i)\bJSON\b`),
		"default expression":      regexp.MustCompile(`(?i)\bDEFAULT\s*\(`),
		"check constraint":        regexp.MustCompile(`(?i)\bCHECK\s*\(`),
	}

	migrationNames := append(mvpMigrationNames(), "050_create_banner_ad.sql")
	for _, name := range migrationNames {
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

func TestPushPreferenceMigrationExtendsPreexistingSchema(t *testing.T) {
	migration := readMigrationFile(t, "052_add_push_preference_message_preview.sql")
	runbook := readMigrationFile(t, "MVP_ROLLBACK.md")

	for _, marker := range []string{
		"ALUMNI_PUSH_PREFERENCE",
		"MESSAGE_PREVIEW_ENABLED",
		"IF NOT EXISTS",
		"ALTER TABLE",
	} {
		if !strings.Contains(migration, marker) {
			t.Errorf("052_add_push_preference_message_preview.sql is missing %q", marker)
		}
	}
	for _, marker := range []string{"pre-existing `ALUMNI_PUSH_PREFERENCE`", "preserve the table", "MESSAGE_PREVIEW_ENABLED"} {
		if !strings.Contains(runbook, marker) {
			t.Fatalf("rollback runbook missing pre-existing push preference guidance %q", marker)
		}
	}
}

func TestMVPMigrationsHaveRollbackRunbookAndNoPublicDonorSchema(t *testing.T) {
	runbook := readMigrationFile(t, "MVP_ROLLBACK.md")
	for _, marker := range []string{"rollback runbook", "backup", "information_schema", "reverse order", "implicit commits"} {
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
	const name = "038_create_auth_principal_tables.sql"
	sql := readMigrationFile(t, name)
	for _, marker := range []string{"USR_STATUS = 'BAA'", "THEN 'rejected'"} {
		if !strings.Contains(sql, marker) {
			t.Errorf("%s does not preserve legacy rejected members: missing %q", name, marker)
		}
	}
}

func TestDonationBackfillFailsClosedOnAmbiguousLegacyRows(t *testing.T) {
	const name = "051_extend_weo_order_for_donation_ledger.sql"
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
			name: "028_create_mobile_device_token_table.sql",
			required: []string{
				"CREATE TABLE IF NOT EXISTS ALUMNI_MOBILE_DEVICE_TOKEN", "MDT_SEQ", "DEVICE_TOKEN",
			},
		},
		{
			name:     "029_extend_mobile_device_token_invalid_state.sql",
			required: []string{"ALUMNI_MOBILE_DEVICE_TOKEN", "INVALID_COUNT", "STATUS"},
		},
		{
			name:     "031_extend_mobile_device_token_apns_metadata.sql",
			required: []string{"ALUMNI_MOBILE_DEVICE_TOKEN", "APNS_ENVIRONMENT", "BUNDLE_ID"},
		},
		{
			name:     "032_allow_android_push_tokens_without_apns_metadata.sql",
			required: []string{"ALUMNI_MOBILE_DEVICE_TOKEN", "APNS_ENVIRONMENT", "BUNDLE_ID"},
		},
		{
			name: "033_backfill_android_push_token_metadata_and_length.sql",
			required: []string{
				"ALUMNI_MOBILE_DEVICE_TOKEN", "VARCHAR(512)", "PLATFORM = 'android'",
			},
		},
		{
			name: "034_create_push_outbox.sql",
			required: []string{
				"ALUMNI_PUSH_OUTBOX", "MDT_SEQ", "DEVICE_TOKEN", "APNS_ENVIRONMENT",
				"PAYLOAD_JSON", "NEXT_ATTEMPT_AT", "ATTEMPT_COUNT",
			},
		},
		{
			name:     "035_create_push_preference.sql",
			required: []string{"ALUMNI_PUSH_PREFERENCE", "NOTICE_ENABLED", "MESSAGE_ENABLED"},
		},
		{
			name: "038_create_auth_principal_tables.sql",
			required: []string{
				"ALUMNI_ADMIN_ROLE", "ALUMNI_VERIFICATION", "unsubmitted",
				"reapproval_pending", "REJECTION_REASON", "REVIEWED_BY",
				"USR_STATUS = 'BAA'", "THEN 'rejected'",
			},
		},
		{
			name: "048_create_member_block_and_message_retention.sql",
			required: []string{
				"ALUMNI_MEMBER_BLOCK", "BLOCKER_USR_SEQ", "BLOCKED_USR_SEQ",
				"AM_CLIENT_MESSAGE_ID", "AM_VISIBLE_RECVR", "AM_SUPPRESSION_REASON", "PURGE_AT",
				"UK_AMB_DIRECTION", "IDX_AMB_BLOCKED", "UK_AM_SENDER_CLIENT",
				"IDX_AM_RECVR_VISIBLE", "IDX_AM_PURGE",
			},
		},
		{
			name: "051_extend_weo_order_for_donation_ledger.sql",
			required: []string{
				"WEO_ORDER", "O_SOURCE", "O_TRANSACTION_NO", "O_COMPOSITE_KEY",
				"O_GROSS_AMOUNT", "O_REFUNDED_AMOUNT", "O_NET_RECEIVED_AMOUNT",
				"O_LIFECYCLE_STATUS", "O_DONATION_DATE",
			},
		},
		{
			name:     "052_add_push_preference_message_preview.sql",
			required: []string{"ALUMNI_PUSH_PREFERENCE", "MESSAGE_PREVIEW_ENABLED"},
		},
		{
			name: "053_create_mobile_app_event_table.sql",
			required: []string{
				"ALUMNI_MOBILE_APP_EVENT", "PLATFORM", "EVENT_TYPE", "USER_ID",
				"APP_VERSION", "OS_VERSION", "DEVICE_MODEL", "OCCURRED_AT", "CREATED_AT",
				"PLATFORM     VARCHAR", "IDX_AME_PLATFORM", "IDX_AME_EVENT_TYPE", "IDX_AME_OCCURRED_AT",
			},
		},
		{
			name: "054_create_app_settings.sql",
			required: []string{
				"CREATE TABLE IF NOT EXISTS app_settings", "AS_KEY", "AS_VALUE",
				"AS_DESCRIPTION", "AS_PUBLIC", "UPDATED_AT", "UPDATED_BY",
				"kakao_open_chat_url", "https://open.kakao.com/o/gNLYTuui",
				"아이디/비밀번호 찾기 문의용 카카오톡 오픈채팅 URL", "'Y'",
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
