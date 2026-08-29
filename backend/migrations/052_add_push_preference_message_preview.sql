-- Migration 052: Add the per-user message preview preference.
-- Target: MariaDB 10.1.38.

DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _052_add_push_preference_message_preview_if_missing()
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_PUSH_PREFERENCE'
          AND COLUMN_NAME = 'MESSAGE_PREVIEW_ENABLED'
    ) THEN
        ALTER TABLE ALUMNI_PUSH_PREFERENCE
            ADD COLUMN MESSAGE_PREVIEW_ENABLED ENUM('Y','N') NOT NULL DEFAULT 'Y'
            AFTER MESSAGE_ENABLED;
    END IF;
END //
DELIMITER ;

CALL _052_add_push_preference_message_preview_if_missing();

DROP PROCEDURE IF EXISTS _052_add_push_preference_message_preview_if_missing;

-- Rollback:
-- ALTER TABLE ALUMNI_PUSH_PREFERENCE DROP COLUMN MESSAGE_PREVIEW_ENABLED;
