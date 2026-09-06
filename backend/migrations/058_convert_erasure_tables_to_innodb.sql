-- Account erasure needs rollback across legacy content, files and donations.
-- MariaDB 10.1.38. Review disk headroom and maintenance time before applying.
-- ALTER TABLE commits implicitly; each conversion is restartable independently.
-- Only known erasure tables that exist and are MyISAM are converted.
DROP PROCEDURE IF EXISTS _058_erasure_storage;
DELIMITER //
CREATE PROCEDURE _058_erasure_storage()
BEGIN
    DECLARE finished INT DEFAULT 0;
    DECLARE target_table VARCHAR(64);
    DECLARE candidates CURSOR FOR
        SELECT TABLE_NAME FROM information_schema.TABLES
        WHERE TABLE_SCHEMA=DATABASE() AND ENGINE='MyISAM'
        AND TABLE_NAME IN (
            'WEO_MEMBER','WEO_MEMBER_LOG','WEO_MEMBER_PUSH','WEO_MEMBER_OUT','WEO_MEMBER_SOCIAL',
            'FUNDAMENTAL_MEMBER','WEO_BOARDBBS','WEO_BOARDCOMAND','WEO_BOARDLIKE','WEO_FILES',
            'WEO_ORDER','WEO_PG_DATA','WEO_PUSH_NOTI','WEO_SMS','WEO_APP_PUSH',
            'WEO_AD_COMMENT','WEO_AD_LIKE','WEO_AD_LOG','WEO_BANNER_AD_LOG',
            'ALUMNI_PUSH_OUTBOX','ALUMNI_MOBILE_DEVICE_TOKEN','ALUMNI_PUSH_DEVICE','ALUMNI_PUSH_PREFERENCE',
            'ALUMNI_MOBILE_REFRESH_TOKEN','ALUMNI_NOTIFICATION','ALUMNI_PASSWORD_RESET','USER_SESSION',
            'ALUMNI_USER_TAG','ALUMNI_ADMIN_ROLE','ALUMNI_VERIFICATION','ALUMNI_MESSAGE',
            'ALUMNI_MESSAGE_REPORT','ALUMNI_MEMBER_BLOCK','ALUMNI_MOBILE_APP_EVENT','WEO_VISIT_DAILY',
            'ALUMNI_SOCIAL_CREDENTIAL','ALUMNI_SOCIAL_REVOCATION_OUTBOX'
        ) ORDER BY TABLE_NAME;
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET finished=1;
    OPEN candidates;
    convert_loop: LOOP
        FETCH candidates INTO target_table;
        IF finished=1 THEN LEAVE convert_loop; END IF;
        SET @erasure_058_ddl=CONCAT('ALTER TABLE `',target_table,'` ENGINE=InnoDB');
        PREPARE erasure_058_statement FROM @erasure_058_ddl;
        EXECUTE erasure_058_statement;
        DEALLOCATE PREPARE erasure_058_statement;
    END LOOP;
    CLOSE candidates;
END//
DELIMITER ;
CALL _058_erasure_storage();
DROP PROCEDURE _058_erasure_storage;
