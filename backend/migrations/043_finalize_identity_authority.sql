-- Migration 043: Finalize canonical identity authority during an approved maintenance window.
-- Application planned: foundation for account unification (email/Kakao/Apple linking).
-- Target: MariaDB 10.1.38. Apply only after migration 042 and a verified T06 backfill run.
-- Application and legacy writers must be frozen before this migration runs.
-- Login-statistics rows in WEO_MEMBER_LOG are retained; removing legacy-session lookup from
-- the application invalidates those legacy cookies without destroying statistics.

SET @_043_latest_run_id = (
    SELECT latest.RUN_ID
    FROM AUTH_IDENTITY_MIGRATION_RUN latest
    ORDER BY latest.STARTED_AT DESC, latest.RUN_ID DESC
    LIMIT 1
);

SET @_043_verified_run_count = (
    SELECT COUNT(*)
    FROM AUTH_IDENTITY_MIGRATION_RUN candidate
    WHERE candidate.RUN_ID = @_043_latest_run_id
      AND candidate.STATUS = 'APPLIED'
      AND candidate.CONFLICT_COUNT = 0
      AND candidate.COMPLETED_AT IS NOT NULL
      AND candidate.SOURCE_FINGERPRINT REGEXP '^[0-9a-f]{64}$'
      AND candidate.SOURCE_FINGERPRINT <> REPEAT('0', 64)
);

SET @_043_journal_step_count = (
    SELECT COUNT(*)
    FROM AUTH_IDENTITY_MIGRATION_JOURNAL journal
    WHERE journal.RUN_ID = @_043_latest_run_id
);

SET @_043_applied_journal_step_count = (
    SELECT COUNT(*)
    FROM AUTH_IDENTITY_MIGRATION_JOURNAL journal
    WHERE journal.RUN_ID = @_043_latest_run_id
      AND journal.STATUS = 'APPLIED'
      AND journal.APPLIED_AT IS NOT NULL
);

SET @_043_guard_sql = IF(
    @_043_verified_run_count = 1
    AND @_043_journal_step_count > 0
    AND @_043_applied_journal_step_count = @_043_journal_step_count,
    'DO 0',
    'SELECT * FROM `_043_verified_canonical_identity_backfill_required`'
);
PREPARE _043_guard_statement FROM @_043_guard_sql;
EXECUTE _043_guard_statement;
DEALLOCATE PREPARE _043_guard_statement;

START TRANSACTION;

UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
SET REVOKED_AT = COALESCE(REVOKED_AT, NOW()),
    MRT_REVOKED_AT = COALESCE(MRT_REVOKED_AT, REVOKED_AT, NOW())
WHERE REVOKED_AT IS NULL
   OR MRT_REVOKED_AT IS NULL;

UPDATE AUTH_SESSION_FAMILY
SET STATUS = 'REVOKED',
    REVOKED_AT = COALESCE(REVOKED_AT, NOW()),
    REVOKE_REASON_CODE = COALESCE(REVOKE_REASON_CODE, 'IDENTITY_AUTHORITY_CUTOVER'),
    UPDATED_AT = NOW()
WHERE STATUS = 'ACTIVE';

COMMIT;

DROP TRIGGER IF EXISTS TRG_WEO_MEMBER_AUTH_PRINCIPAL_INSERT;
DROP TRIGGER IF EXISTS TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE;
