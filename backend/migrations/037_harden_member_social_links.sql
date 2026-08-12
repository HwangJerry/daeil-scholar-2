-- Migration 037: Canonical social-link metadata and one-link-per-provider invariant.
-- Target: MariaDB 10.1.38. Apply after a backup.
-- Preflight stops before table ALTER when duplicate links, conflicting owners,
-- or unsupported legacy status values exist.

DELIMITER //
DROP PROCEDURE IF EXISTS _037_add_column_if_missing //
DROP PROCEDURE IF EXISTS _037_require_preflight //
DROP PROCEDURE IF EXISTS _037_add_unique_index_if_missing //

CREATE PROCEDURE _037_add_column_if_missing(
    IN p_table VARCHAR(64),
    IN p_column VARCHAR(64),
    IN p_definition TEXT
)
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = p_table
          AND COLUMN_NAME = p_column
    ) THEN
        SET @ddl = CONCAT('ALTER TABLE ', p_table, ' ADD COLUMN ', p_column, ' ', p_definition);
        PREPARE stmt FROM @ddl;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END //

CREATE PROCEDURE _037_require_preflight()
BEGIN
    IF EXISTS (
        SELECT 1
        FROM WEO_MEMBER_SOCIAL
        GROUP BY USR_SEQ, NMS_GATE
        HAVING COUNT(*) > 1
    ) THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'duplicate WEO_MEMBER_SOCIAL user/provider links must be resolved';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM WEO_MEMBER_SOCIAL
        GROUP BY NMS_GATE, NMS_ID
        HAVING COUNT(DISTINCT USR_SEQ) > 1
    ) THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'conflicting WEO_MEMBER_SOCIAL provider/subject owners must be resolved';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM WEO_MEMBER_SOCIAL
        WHERE NMS_STATUS IS NULL
           OR NMS_STATUS NOT IN ('Y', 'N', 'ACTIVE', 'INACTIVE')
    ) THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'unsupported WEO_MEMBER_SOCIAL status must be resolved';
    END IF;
END //

CREATE PROCEDURE _037_add_unique_index_if_missing(
    IN p_index VARCHAR(64),
    IN p_columns VARCHAR(255)
)
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM (
            SELECT INDEX_NAME
            FROM information_schema.STATISTICS
            WHERE TABLE_SCHEMA = DATABASE()
              AND TABLE_NAME = 'WEO_MEMBER_SOCIAL'
              AND NON_UNIQUE = 0
            GROUP BY INDEX_NAME
            HAVING GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) = p_columns
        ) AS matching_unique_index
    ) THEN
        SET @ddl = CONCAT(
            'ALTER TABLE WEO_MEMBER_SOCIAL ADD UNIQUE KEY ',
            p_index,
            ' (',
            p_columns,
            ')'
        );
        PREPARE stmt FROM @ddl;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END //
DELIMITER ;

CALL _037_require_preflight();

CALL _037_add_column_if_missing(
    'WEO_MEMBER_SOCIAL',
    'NMS_EMAIL',
    'VARCHAR(255) NULL AFTER NMS_ID'
);
CALL _037_add_column_if_missing(
    'WEO_MEMBER_SOCIAL',
    'NMS_STATUS',
    "VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' AFTER NMS_EMAIL"
);
CALL _037_add_column_if_missing(
    'WEO_MEMBER_SOCIAL',
    'NMS_EMAIL_ENABLED',
    "ENUM('Y','N') NOT NULL DEFAULT 'Y' AFTER NMS_STATUS"
);

ALTER TABLE WEO_MEMBER_SOCIAL
    MODIFY NMS_EMAIL VARCHAR(255) NULL,
    MODIFY NMS_STATUS VARCHAR(20) NULL DEFAULT 'ACTIVE';

UPDATE WEO_MEMBER_SOCIAL
SET NMS_STATUS = 'ACTIVE'
WHERE NMS_STATUS = 'Y';

ALTER TABLE WEO_MEMBER_SOCIAL
    ENGINE=InnoDB,
    MODIFY NMS_STATUS VARCHAR(20) NOT NULL DEFAULT 'ACTIVE';

CALL _037_add_unique_index_if_missing('UK_USR_PROVIDER', 'USR_SEQ,NMS_GATE');
CALL _037_add_unique_index_if_missing('UK_PROVIDER_SUBJECT', 'NMS_GATE,NMS_ID');

DROP PROCEDURE _037_add_unique_index_if_missing;
DROP PROCEDURE _037_require_preflight;
DROP PROCEDURE _037_add_column_if_missing;
