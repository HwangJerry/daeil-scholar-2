-- Restore guard: erased member numbers kept only while backups may still hold them.
-- MariaDB 10.1.38. A row expires 35 days after completion (28-day backup
-- retention plus one weekly backup interval) and is purged by the erasure job.
-- After restoring a backup, cmd/erasure-restore-reapply erases these members again.
CREATE TABLE IF NOT EXISTS ALUMNI_ERASURE_RESTORE_GUARD (
    REQUEST_ID   BIGINT NOT NULL PRIMARY KEY,
    USR_SEQ      INT NOT NULL,
    COMPLETED_AT DATETIME NOT NULL,
    EXPIRES_AT   DATETIME NOT NULL,
    KEY IDX_RESTORE_GUARD_EXPIRY (EXPIRES_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Rollback (only after reverting the application binary):
-- DROP TABLE ALUMNI_ERASURE_RESTORE_GUARD;
