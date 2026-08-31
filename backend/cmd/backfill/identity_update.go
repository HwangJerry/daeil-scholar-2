// Identity updates — persists migration audit rows and canonical identity records.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

const duplicateEntryErrorNumber uint16 = 1062

type identityExecContext interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func insertIdentityMigrationRun(ctx context.Context, executor identityExecContext, runID, fingerprint string) error {
	_, err := executor.ExecContext(ctx, `
		INSERT INTO AUTH_IDENTITY_MIGRATION_RUN (
			RUN_ID, STATUS, SOURCE_FINGERPRINT, CONFLICT_COUNT,
			FAILURE_CODE, STARTED_AT, COMPLETED_AT, UPDATED_AT
		) VALUES (?, 'APPLYING', ?, 0, NULL, NOW(), NULL, NOW())
	`, runID, fingerprint)
	return err
}

func markIdentityMigrationRunApplied(ctx context.Context, executor identityExecContext, runID, fingerprint string) error {
	result, err := executor.ExecContext(ctx, `
		UPDATE AUTH_IDENTITY_MIGRATION_RUN
		SET STATUS = 'APPLIED',
			SOURCE_FINGERPRINT = ?,
			CONFLICT_COUNT = 0,
			FAILURE_CODE = NULL,
			COMPLETED_AT = NOW(),
			UPDATED_AT = NOW()
		WHERE RUN_ID = ? AND STATUS = 'APPLYING'
	`, fingerprint, runID)
	return requireOneAffectedRow(result, err, "mark identity migration run applied")
}

func markIdentityMigrationRunFailed(ctx context.Context, executor identityExecContext, runID string, conflictCount int, failureCode string) error {
	result, err := executor.ExecContext(ctx, `
		UPDATE AUTH_IDENTITY_MIGRATION_RUN
		SET STATUS = 'FAILED',
			CONFLICT_COUNT = ?,
			FAILURE_CODE = ?,
			COMPLETED_AT = NOW(),
			UPDATED_AT = NOW()
		WHERE RUN_ID = ? AND STATUS = 'APPLYING'
	`, conflictCount, failureCode, runID)
	return requireOneAffectedRow(result, err, "mark identity migration run failed")
}

func startIdentityJournalStep(ctx context.Context, executor identityExecContext, runID, stepKey string) error {
	_, err := executor.ExecContext(ctx, `
		INSERT INTO AUTH_IDENTITY_MIGRATION_JOURNAL (
			RUN_ID, STEP_KEY, STATUS, FAILURE_CODE,
			STARTED_AT, APPLIED_AT, UPDATED_AT
		) VALUES (?, ?, 'STARTED', NULL, NOW(), NULL, NOW())
	`, runID, stepKey)
	return err
}

func applyIdentityJournalStep(ctx context.Context, executor identityExecContext, runID, stepKey string) error {
	result, err := executor.ExecContext(ctx, `
		UPDATE AUTH_IDENTITY_MIGRATION_JOURNAL
		SET STATUS = 'APPLIED', APPLIED_AT = NOW(), UPDATED_AT = NOW()
		WHERE RUN_ID = ? AND STEP_KEY = ? AND STATUS = 'STARTED'
	`, runID, stepKey)
	return requireOneAffectedRow(result, err, "mark identity journal step applied")
}

func insertCanonicalAccountState(ctx context.Context, executor identityExecContext, member identityMemberRow) error {
	state := mapLegacyAccountStatus(member.LegacyStatus)
	_, err := executor.ExecContext(ctx, `
		INSERT INTO AUTH_ACCOUNT_STATE (
			ACCOUNT_ID, STATUS, SUSPENDED_AT, WITHDRAWN_AT, CREATED_AT, UPDATED_AT
		) VALUES (
			?, ?,
			CASE WHEN ? THEN NOW() ELSE NULL END,
			CASE WHEN ? THEN NOW() ELSE NULL END,
			NOW(), NOW()
		)
	`, member.AccountID, state.Status, state.SuspendedAt, state.WithdrawnAt)
	return err
}

func insertCanonicalPasswordIdentity(
	ctx context.Context,
	executor identityExecContext,
	member identityMemberRow,
	email *string,
) (int64, error) {
	result, err := executor.ExecContext(ctx, `
		INSERT INTO AUTH_IDENTITY (
			ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL, STATUS,
			VERIFIED_AT, CREATED_AT, UPDATED_AT
		) VALUES (?, ?, ?, ?, 'ACTIVE', NOW(), NOW(), NOW())
	`, member.AccountID, identityProviderLocalUsername, member.Username, email)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func insertCanonicalPasswordCredential(
	ctx context.Context,
	executor identityExecContext,
	identityID int64,
	passwordHash string,
) error {
	_, err := executor.ExecContext(ctx, `
		INSERT INTO AUTH_PASSWORD_CREDENTIAL (
			IDENTITY_ID, PROVIDER, ALGORITHM, PARAMETERS_TEXT,
			PASSWORD_HASH, STATUS, CREATED_AT, UPDATED_AT
		) VALUES (?, ?, ?, NULL, ?, 'ACTIVE', NOW(), NOW())
	`, identityID, identityProviderLocalUsername, legacyPasswordAlgorithm, passwordHash)
	return err
}

func insertCanonicalSocialIdentity(
	ctx context.Context,
	executor identityExecContext,
	link identitySocialRow,
	provider string,
	status string,
) error {
	_, err := executor.ExecContext(ctx, `
		INSERT INTO AUTH_IDENTITY (
			ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL, STATUS,
			VERIFIED_AT, CREATED_AT, UPDATED_AT
		) VALUES (?, ?, ?, NULL, ?, NULL, NOW(), NOW())
	`, link.AccountID, provider, link.SubjectKey, status)
	return err
}

func classifyUniqueConstraintError(err error, fallbackReason string) (string, bool) {
	var mysqlError *mysql.MySQLError
	if !errors.As(err, &mysqlError) || mysqlError.Number != duplicateEntryErrorNumber {
		return "", false
	}
	message := strings.ToLower(mysqlError.Message)
	switch {
	case strings.Contains(message, "uq_auth_identity_provider_subject"):
		return "duplicate_provider_subject", true
	case strings.Contains(message, "uq_auth_identity_normalized_email"):
		return "duplicate_normalized_email", true
	default:
		return fallbackReason, true
	}
}

func requireOneAffectedRow(result sql.Result, err error, operation string) error {
	if err != nil {
		return err
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows affected: %w", operation, err)
	}
	if affectedRows != 1 {
		return fmt.Errorf("%s affected %d rows, want 1", operation, affectedRows)
	}
	return nil
}
