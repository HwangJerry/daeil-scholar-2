package repository

import "github.com/jmoiron/sqlx"

const canonicalPasswordRequiredTableCount = 5

// CanonicalPasswordWriteReady prevents application cutover before the
// canonical schema and a conflict-free, fully journaled backfill exist.
func CanonicalPasswordWriteReady(db *sqlx.DB) (bool, error) {
	var tableCount int
	if err := db.Get(&tableCount, `
		SELECT COUNT(*)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME IN (
		    'AUTH_IDENTITY',
		    'AUTH_PASSWORD_CREDENTIAL',
		    'AUTH_IDENTITY_MIGRATION_RUN',
		    'AUTH_IDENTITY_MIGRATION_JOURNAL',
		    '_migration_history'
		  )
	`); err != nil {
		return false, err
	}
	if tableCount != canonicalPasswordRequiredTableCount {
		return false, nil
	}

	var readyCount int
	if err := db.Get(&readyCount, `
		SELECT COUNT(*)
		FROM AUTH_IDENTITY_MIGRATION_RUN run
		WHERE run.RUN_ID = (
			SELECT latest.RUN_ID
			FROM AUTH_IDENTITY_MIGRATION_RUN latest
			ORDER BY latest.STARTED_AT DESC, latest.RUN_ID DESC
			LIMIT 1
		)
		  AND run.STATUS = 'APPLIED'
		  AND run.CONFLICT_COUNT = 0
		  AND run.COMPLETED_AT IS NOT NULL
		  AND run.SOURCE_FINGERPRINT REGEXP '^[0-9a-f]{64}$'
		  AND run.SOURCE_FINGERPRINT <> REPEAT('0', 64)
		  AND EXISTS (
			SELECT 1 FROM AUTH_IDENTITY_MIGRATION_JOURNAL journal
			WHERE journal.RUN_ID = run.RUN_ID
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM AUTH_IDENTITY_MIGRATION_JOURNAL journal
			WHERE journal.RUN_ID = run.RUN_ID
			  AND (journal.STATUS <> 'APPLIED' OR journal.APPLIED_AT IS NULL)
		  )
		  AND EXISTS (
			SELECT 1 FROM _migration_history
			WHERE filename = '043_finalize_identity_authority.sql'
		  )
		  AND NOT EXISTS (
			SELECT 1
			FROM information_schema.TRIGGERS
			WHERE TRIGGER_SCHEMA = DATABASE()
			  AND TRIGGER_NAME IN (
				'TRG_WEO_MEMBER_AUTH_PRINCIPAL_INSERT',
				'TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE'
			  )
		  )
	`); err != nil {
		return false, err
	}
	return readyCount == 1, nil
}
