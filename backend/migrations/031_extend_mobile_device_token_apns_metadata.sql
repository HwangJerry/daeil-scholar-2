-- 031_extend_mobile_device_token_apns_metadata.sql — APNs routing metadata per device token

DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _add_mobile_token_apns_metadata_if_missing()
BEGIN
    SET @col_exists = 0;
    SELECT COUNT(*) INTO @col_exists
      FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'ALUMNI_MOBILE_DEVICE_TOKEN'
       AND COLUMN_NAME = 'APNS_ENVIRONMENT';
    IF @col_exists = 0 THEN
        ALTER TABLE ALUMNI_MOBILE_DEVICE_TOKEN
            ADD COLUMN APNS_ENVIRONMENT ENUM('sandbox','production') NOT NULL DEFAULT 'production' AFTER DEVICE_TOKEN;
    END IF;

    SET @col_exists = 0;
    SELECT COUNT(*) INTO @col_exists
      FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'ALUMNI_MOBILE_DEVICE_TOKEN'
       AND COLUMN_NAME = 'BUNDLE_ID';
    IF @col_exists = 0 THEN
        ALTER TABLE ALUMNI_MOBILE_DEVICE_TOKEN
            ADD COLUMN BUNDLE_ID VARCHAR(255) NULL AFTER APNS_ENVIRONMENT;
    END IF;
END //
DELIMITER ;

CALL _add_mobile_token_apns_metadata_if_missing();

DROP PROCEDURE IF EXISTS _add_mobile_token_apns_metadata_if_missing;
