-- Migration 032: Mobile push devices and account-wide notification preferences.
-- Target: MariaDB 10.1.38. Apply manually after a backup.
-- Device tokens use an ASCII binary collation so the unique key stays within
-- the legacy InnoDB index byte limit and token comparisons remain exact.

DROP PROCEDURE IF EXISTS _add_column_if_not_exists;

DELIMITER $$

CREATE PROCEDURE _add_column_if_not_exists(
    IN p_table VARCHAR(64),
    IN p_column VARCHAR(64),
    IN p_definition TEXT
)
BEGIN
    SET @column_exists = 0;
    SELECT COUNT(*) INTO @column_exists
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = p_table
      AND COLUMN_NAME = p_column;

    IF @column_exists = 0 THEN
        SET @ddl = CONCAT(
            'ALTER TABLE `', p_table, '` ADD COLUMN `', p_column, '` ', p_definition
        );
        PREPARE statement FROM @ddl;
        EXECUTE statement;
        DEALLOCATE PREPARE statement;
    END IF;
END$$

DELIMITER ;

CREATE TABLE IF NOT EXISTS ALUMNI_PUSH_DEVICE (
    APD_SEQ            BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    USR_SEQ            INT NOT NULL,
    PLATFORM           ENUM('android','ios') NOT NULL,
    DEVICE_TOKEN       VARCHAR(512) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    APNS_ENVIRONMENT   ENUM('sandbox','production') NULL,
    BUNDLE_ID          VARCHAR(255) NULL,
    LOCALE             VARCHAR(20) NULL,
    LAST_SEEN_AT       DATETIME NOT NULL,
    CREATED_AT         DATETIME NOT NULL,
    UPDATED_AT         DATETIME NOT NULL,
    UNIQUE KEY UK_APD_PLATFORM_TOKEN (PLATFORM, DEVICE_TOKEN),
    INDEX IDX_APD_USER (USR_SEQ, PLATFORM),
    INDEX IDX_APD_LAST_SEEN (LAST_SEEN_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_PUSH_PREFERENCE (
    USR_SEQ                   INT NOT NULL PRIMARY KEY,
    MESSAGE_ENABLED           ENUM('Y','N') NOT NULL DEFAULT 'Y',
    MESSAGE_PREVIEW_ENABLED   ENUM('Y','N') NOT NULL DEFAULT 'Y',
    CREATED_AT                DATETIME NOT NULL,
    UPDATED_AT                DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CALL _add_column_if_not_exists(
    'ALUMNI_PUSH_PREFERENCE',
    'MESSAGE_PREVIEW_ENABLED',
    'ENUM(''Y'',''N'') NOT NULL DEFAULT ''Y'' AFTER MESSAGE_ENABLED'
);

DROP PROCEDURE IF EXISTS _add_column_if_not_exists;
