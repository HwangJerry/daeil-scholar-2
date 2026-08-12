-- 029_extend_mobile_device_token_invalid_state.sql — Persist invalid push token state

DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _add_mobile_token_invalid_count_if_missing()
BEGIN
    SET @col_exists = 0;
    SELECT COUNT(*) INTO @col_exists
      FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'ALUMNI_MOBILE_DEVICE_TOKEN'
       AND COLUMN_NAME = 'INVALID_COUNT';
    IF @col_exists = 0 THEN
        ALTER TABLE ALUMNI_MOBILE_DEVICE_TOKEN
            ADD COLUMN INVALID_COUNT INT NOT NULL DEFAULT 0 AFTER STATUS;
    END IF;
END //
DELIMITER ;

CALL _add_mobile_token_invalid_count_if_missing();

ALTER TABLE ALUMNI_MOBILE_DEVICE_TOKEN
    MODIFY STATUS ENUM('ACTIVE','INACTIVE','STALE','UNVERIFIED','REVOKED') DEFAULT 'ACTIVE';

DROP PROCEDURE IF EXISTS _add_mobile_token_invalid_count_if_missing;
