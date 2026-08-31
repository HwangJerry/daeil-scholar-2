// Identity backfill — orchestrates the audited canonical identity migration run.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const (
	accountStateStep       = "account_state"
	passwordIdentitiesStep = "password_identities"
	socialIdentitiesStep   = "social_identities"

	failureCodeConflicts = "SOURCE_CONFLICTS"
	failureCodeExecution = "BACKFILL_EXECUTION_FAILED"
)

var errIdentityBackfillConflicts = errors.New("canonical identity backfill found conflicts")

type identityBackfillOptions struct {
	DryRun bool
}

type identityBackfillSource struct {
	Members     []identityMemberRow
	SocialLinks []identitySocialRow
	Fingerprint string
}

func BackfillIdentities(
	ctx context.Context,
	db *sqlx.DB,
	options identityBackfillOptions,
) (identityBackfillStats, error) {
	source, err := loadIdentityBackfillSource(ctx, db)
	if err != nil {
		return identityBackfillStats{}, err
	}

	runID := uuid.NewString()
	if options.DryRun {
		return dryRunIdentityBackfill(ctx, db, runID, source)
	}
	return applyIdentityBackfill(ctx, db, runID, source)
}

func loadIdentityBackfillSource(ctx context.Context, db *sqlx.DB) (identityBackfillSource, error) {
	members, err := fetchIdentityMembers(ctx, db)
	if err != nil {
		return identityBackfillSource{}, err
	}
	socialLinks, err := fetchIdentitySocialLinks(ctx, db)
	if err != nil {
		return identityBackfillSource{}, err
	}
	memberCount, maxAccountID := sourceSummary(members)
	return identityBackfillSource{
		Members:     members,
		SocialLinks: socialLinks,
		Fingerprint: sourceFingerprint(memberCount, maxAccountID),
	}, nil
}

func dryRunIdentityBackfill(
	ctx context.Context,
	db *sqlx.DB,
	runID string,
	source identityBackfillSource,
) (identityBackfillStats, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return identityBackfillStats{}, fmt.Errorf("begin identity dry-run transaction: %w", err)
	}
	defer tx.Rollback()

	if err := insertIdentityMigrationRun(ctx, tx, runID, source.Fingerprint); err != nil {
		return identityBackfillStats{}, fmt.Errorf("create identity dry-run audit row: %w", err)
	}
	stats, err := executeIdentityBackfill(ctx, tx, runID, source)
	if err != nil {
		return stats, err
	}
	if err := tx.Rollback(); err != nil {
		return stats, fmt.Errorf("rollback identity dry-run transaction: %w", err)
	}

	log.Printf("identity backfill dry-run: run_id=%s fingerprint=%s %s", runID, source.Fingerprint, stats)
	if stats.ConflictCount > 0 {
		return stats, fmt.Errorf("%w: count=%d", errIdentityBackfillConflicts, stats.ConflictCount)
	}
	return stats, nil
}

func applyIdentityBackfill(
	ctx context.Context,
	db *sqlx.DB,
	runID string,
	source identityBackfillSource,
) (identityBackfillStats, error) {
	if err := insertIdentityMigrationRun(ctx, db, runID, source.Fingerprint); err != nil {
		return identityBackfillStats{}, fmt.Errorf("create identity migration run: %w", err)
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return identityBackfillStats{}, failIdentityRun(ctx, db, runID, 0, fmt.Errorf("begin identity transaction: %w", err))
	}
	defer tx.Rollback()

	stats, err := executeIdentityBackfill(ctx, tx, runID, source)
	if err != nil {
		_ = tx.Rollback()
		return stats, failIdentityRun(ctx, db, runID, stats.ConflictCount, err)
	}
	if stats.ConflictCount > 0 {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return stats, fmt.Errorf("rollback conflicting identity backfill: %w", rollbackErr)
		}
		if err := markIdentityMigrationRunFailed(ctx, db, runID, stats.ConflictCount, failureCodeConflicts); err != nil {
			return stats, fmt.Errorf("mark conflicting identity run failed: %w", err)
		}
		log.Printf("identity backfill failed: run_id=%s fingerprint=%s %s", runID, source.Fingerprint, stats)
		return stats, fmt.Errorf("%w: count=%d", errIdentityBackfillConflicts, stats.ConflictCount)
	}

	if err := markIdentityMigrationRunApplied(ctx, tx, runID, source.Fingerprint); err != nil {
		_ = tx.Rollback()
		return stats, failIdentityRun(ctx, db, runID, 0, fmt.Errorf("mark identity migration run applied: %w", err))
	}
	if err := tx.Commit(); err != nil {
		// A commit error has an unknown outcome; do not overwrite a possibly
		// committed APPLIED run with a separate autocommit update.
		return stats, fmt.Errorf("commit identity backfill (outcome unknown): %w", err)
	}

	log.Printf("identity backfill applied: run_id=%s fingerprint=%s %s", runID, source.Fingerprint, stats)
	return stats, nil
}

func executeIdentityBackfill(
	ctx context.Context,
	tx *sqlx.Tx,
	runID string,
	source identityBackfillSource,
) (identityBackfillStats, error) {
	stats := identityBackfillStats{
		MembersScanned:     len(source.Members),
		SocialLinksScanned: len(source.SocialLinks),
	}

	if err := backfillAccountStates(ctx, tx, runID, source.Members, &stats); err != nil {
		return stats, err
	}
	if err := backfillPasswordIdentities(ctx, tx, runID, source.Members, &stats); err != nil {
		return stats, err
	}
	if err := backfillSocialIdentities(ctx, tx, runID, source.Members, source.SocialLinks, &stats); err != nil {
		return stats, err
	}
	return stats, nil
}

func backfillAccountStates(
	ctx context.Context,
	tx *sqlx.Tx,
	runID string,
	members []identityMemberRow,
	stats *identityBackfillStats,
) error {
	if err := startIdentityJournalStep(ctx, tx, runID, accountStateStep); err != nil {
		return fmt.Errorf("start %s journal: %w", accountStateStep, err)
	}
	for _, member := range members {
		if err := insertCanonicalAccountState(ctx, tx, member); err != nil {
			if recordIdentityWriteConflict(stats, accountStateStep, member.AccountID, "duplicate_account_state", err) {
				continue
			}
			return fmt.Errorf("insert account state for USR_SEQ %d: %w", member.AccountID, err)
		}
		stats.AccountStatesCreated++
	}
	if err := applyIdentityJournalStep(ctx, tx, runID, accountStateStep); err != nil {
		return fmt.Errorf("apply %s journal: %w", accountStateStep, err)
	}
	return nil
}

func backfillPasswordIdentities(
	ctx context.Context,
	tx *sqlx.Tx,
	runID string,
	members []identityMemberRow,
	stats *identityBackfillStats,
) error {
	if err := startIdentityJournalStep(ctx, tx, runID, passwordIdentitiesStep); err != nil {
		return fmt.Errorf("start %s journal: %w", passwordIdentitiesStep, err)
	}
	for _, member := range members {
		if mapLegacyAccountStatus(member.LegacyStatus).Status == accountStatusWithdrawn {
			continue
		}
		if !hasLegacyPassword(member.PasswordHash) {
			continue
		}
		if !isValidIdentityKey(member.Username) {
			recordIdentitySourceConflict(stats, passwordIdentitiesStep, member.AccountID, "invalid_local_username")
			continue
		}
		if !isMysqlNativePasswordHash(member.PasswordHash) {
			recordIdentitySourceConflict(stats, passwordIdentitiesStep, member.AccountID, "unreadable_password_algorithm")
			continue
		}
		email := normalizedEmail(member.Email)
		if !isValidNormalizedEmail(email) {
			recordIdentitySourceConflict(stats, passwordIdentitiesStep, member.AccountID, "invalid_normalized_email")
			continue
		}

		identityID, err := insertCanonicalPasswordIdentity(ctx, tx, member, email)
		if err != nil {
			if recordIdentityWriteConflict(stats, passwordIdentitiesStep, member.AccountID, "duplicate_password_identity", err) {
				continue
			}
			return fmt.Errorf("insert password identity for USR_SEQ %d: %w", member.AccountID, err)
		}
		stats.PasswordIdentitiesCreated++
		if err := insertCanonicalPasswordCredential(ctx, tx, identityID, member.PasswordHash); err != nil {
			if recordIdentityWriteConflict(stats, passwordIdentitiesStep, member.AccountID, "duplicate_password_credential", err) {
				continue
			}
			return fmt.Errorf("insert password credential for USR_SEQ %d: %w", member.AccountID, err)
		}
		stats.PasswordCredentialsCreated++
	}
	if err := applyIdentityJournalStep(ctx, tx, runID, passwordIdentitiesStep); err != nil {
		return fmt.Errorf("apply %s journal: %w", passwordIdentitiesStep, err)
	}
	return nil
}

func backfillSocialIdentities(
	ctx context.Context,
	tx *sqlx.Tx,
	runID string,
	members []identityMemberRow,
	links []identitySocialRow,
	stats *identityBackfillStats,
) error {
	if err := startIdentityJournalStep(ctx, tx, runID, socialIdentitiesStep); err != nil {
		return fmt.Errorf("start %s journal: %w", socialIdentitiesStep, err)
	}
	accountIDs := accountIDSet(members)
	for _, link := range links {
		if _, exists := accountIDs[link.AccountID]; !exists {
			recordIdentitySourceConflict(stats, socialIdentitiesStep, link.AccountID, "orphan_social_link")
			continue
		}
		provider, providerValid := mapLegacySocialProvider(link.LegacyProvider)
		if !providerValid {
			recordIdentitySourceConflict(stats, socialIdentitiesStep, link.AccountID, "unsupported_social_provider")
			continue
		}
		if !isValidIdentityKey(link.SubjectKey) {
			recordIdentitySourceConflict(stats, socialIdentitiesStep, link.AccountID, "invalid_social_subject")
			continue
		}
		status, statusValid := mapLegacySocialStatus(link.LegacyStatus)
		if !statusValid {
			recordIdentitySourceConflict(stats, socialIdentitiesStep, link.AccountID, "unsupported_social_status")
			continue
		}
		if err := insertCanonicalSocialIdentity(ctx, tx, link, provider, status); err != nil {
			if recordIdentityWriteConflict(stats, socialIdentitiesStep, link.AccountID, "duplicate_social_identity", err) {
				continue
			}
			return fmt.Errorf("insert social identity for USR_SEQ %d: %w", link.AccountID, err)
		}
		stats.SocialIdentitiesCreated++
	}
	if err := applyIdentityJournalStep(ctx, tx, runID, socialIdentitiesStep); err != nil {
		return fmt.Errorf("apply %s journal: %w", socialIdentitiesStep, err)
	}
	return nil
}

func recordIdentityWriteConflict(
	stats *identityBackfillStats,
	step string,
	accountID int,
	fallbackReason string,
	err error,
) bool {
	reason, isConflict := classifyUniqueConstraintError(err, fallbackReason)
	if !isConflict {
		return false
	}
	recordIdentitySourceConflict(stats, step, accountID, reason)
	return true
}

func recordIdentitySourceConflict(stats *identityBackfillStats, step string, accountID int, reason string) {
	stats.ConflictCount = incrementConflictCount(stats.ConflictCount, true)
	log.Printf("identity backfill conflict: step=%s usr_seq=%d reason=%s", step, accountID, reason)
}

func failIdentityRun(ctx context.Context, db *sqlx.DB, runID string, conflictCount int, cause error) error {
	if err := markIdentityMigrationRunFailed(ctx, db, runID, conflictCount, failureCodeExecution); err != nil {
		return errors.Join(cause, fmt.Errorf("mark identity migration run failed: %w", err))
	}
	return cause
}
