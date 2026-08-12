-- Migration 036: Extend the production mobile refresh-token table in place.
-- Target: MariaDB 10.1.38. Apply after a backup.
-- Existing MRT_REVOKED_AT remains intact for rollback compatibility.

DELIMITER //
CREATE PROCEDURE _036_add_column_if_missing(
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

CREATE PROCEDURE _036_add_index_if_missing(
    IN p_table VARCHAR(64),
    IN p_index VARCHAR(64),
    IN p_columns TEXT
)
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = p_table
          AND INDEX_NAME = p_index
    ) THEN
        SET @ddl = CONCAT('CREATE INDEX ', p_index, ' ON ', p_table, ' (', p_columns, ')');
        PREPARE stmt FROM @ddl;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END //
DELIMITER ;

CALL _036_add_column_if_missing(
    'ALUMNI_MOBILE_REFRESH_TOKEN',
    'CONSUMED_AT',
    'DATETIME NULL AFTER CREATED_AT'
);
CALL _036_add_column_if_missing(
    'ALUMNI_MOBILE_REFRESH_TOKEN',
    'REVOKED_AT',
    'DATETIME NULL AFTER CONSUMED_AT'
);
CALL _036_add_column_if_missing(
    'ALUMNI_MOBILE_REFRESH_TOKEN',
    'ROTATED_TO_JTI',
    'VARCHAR(64) NULL AFTER REVOKED_AT'
);
CALL _036_add_index_if_missing(
    'ALUMNI_MOBILE_REFRESH_TOKEN',
    'IDX_MRT_SID',
    'MRT_SID'
);

UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
SET REVOKED_AT = MRT_REVOKED_AT
WHERE REVOKED_AT IS NULL
  AND MRT_REVOKED_AT IS NOT NULL;

DROP PROCEDURE _036_add_index_if_missing;
DROP PROCEDURE _036_add_column_if_missing;
