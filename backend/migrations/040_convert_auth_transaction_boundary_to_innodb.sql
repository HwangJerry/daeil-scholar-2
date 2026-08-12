-- Migration 040: Make the canonical account row transactional.
-- Target: MariaDB 10.1.38.
--
-- WEO_MEMBER is the canonical account authority. Its legacy MyISAM engine
-- commits independently from the InnoDB identity/verification children written
-- in the same account-creation transaction. Converting this table closes that
-- mixed-engine transaction boundary.

SET @t03_current_engine = (
    SELECT ENGINE
    FROM information_schema.TABLES
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'WEO_MEMBER'
      AND TABLE_TYPE = 'BASE TABLE'
    LIMIT 1
);

SET @t03_validate_sql = IF(
    @t03_current_engine IN ('MyISAM', 'InnoDB'),
    'DO 0',
    'SELECT * FROM information_schema._040_invalid_weo_member_engine'
);
PREPARE t03_validate_statement FROM @t03_validate_sql;
EXECUTE t03_validate_statement;
DEALLOCATE PREPARE t03_validate_statement;

SET @t03_convert_sql = IF(
    @t03_current_engine = 'MyISAM',
    'ALTER TABLE WEO_MEMBER ENGINE=InnoDB',
    'DO 0'
);
PREPARE t03_convert_statement FROM @t03_convert_sql;
EXECUTE t03_convert_statement;
DEALLOCATE PREPARE t03_convert_statement;

SET @t03_current_engine = NULL;
SET @t03_validate_sql = NULL;
SET @t03_convert_sql = NULL;
