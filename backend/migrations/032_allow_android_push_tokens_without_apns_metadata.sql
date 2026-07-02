-- 032_allow_android_push_tokens_without_apns_metadata.sql — Android FCM tokens do not carry APNs routing metadata

DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _allow_nullable_mobile_token_apns_metadata()
BEGIN
    SET @col_exists = 0;
    SELECT COUNT(*) INTO @col_exists
      FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'ALUMNI_MOBILE_DEVICE_TOKEN'
       AND COLUMN_NAME = 'APNS_ENVIRONMENT';
    IF @col_exists > 0 THEN
        ALTER TABLE ALUMNI_MOBILE_DEVICE_TOKEN
            MODIFY APNS_ENVIRONMENT ENUM('sandbox','production') NULL DEFAULT NULL;
    END IF;

    SET @col_exists = 0;
    SELECT COUNT(*) INTO @col_exists
      FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'ALUMNI_MOBILE_DEVICE_TOKEN'
       AND COLUMN_NAME = 'BUNDLE_ID';
    IF @col_exists > 0 THEN
        ALTER TABLE ALUMNI_MOBILE_DEVICE_TOKEN
            MODIFY BUNDLE_ID VARCHAR(255) NULL DEFAULT NULL;
    END IF;
END //
DELIMITER ;

CALL _allow_nullable_mobile_token_apns_metadata();

DROP PROCEDURE IF EXISTS _allow_nullable_mobile_token_apns_metadata;
