-- Migration 076: Add the per-user notice (새 소식) push preference.
-- ALUMNI_PUSH_PREFERENCE was created with NOTICE_ENABLED in 035, but databases
-- that were provisioned from a lineage without it must still gain the column
-- before the notice fan-out can filter on it. The guard makes this a no-op
-- wherever the column is already present.
-- Target: MariaDB 10.1.38.

DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _076_add_push_preference_notice_if_missing()
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_PUSH_PREFERENCE'
          AND COLUMN_NAME = 'NOTICE_ENABLED'
    ) THEN
        ALTER TABLE ALUMNI_PUSH_PREFERENCE
            ADD COLUMN NOTICE_ENABLED ENUM('Y','N') NOT NULL DEFAULT 'Y'
            AFTER MESSAGE_PREVIEW_ENABLED;
    END IF;
END //
DELIMITER ;

CALL _076_add_push_preference_notice_if_missing();

DROP PROCEDURE IF EXISTS _076_add_push_preference_notice_if_missing;

-- Rollback:
-- ALTER TABLE ALUMNI_PUSH_PREFERENCE DROP COLUMN NOTICE_ENABLED;
