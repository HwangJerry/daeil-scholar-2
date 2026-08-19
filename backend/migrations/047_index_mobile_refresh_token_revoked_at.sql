-- Migration 047: Add the cleanup lookup index for the active refresh-token
-- revocation timestamp. The existing IDX_REVOKED index covers only the legacy
-- MRT_REVOKED_AT column.
-- Target: MariaDB 10.1.38.

DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _047_add_mobile_refresh_token_revoked_index_if_missing()
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_MOBILE_REFRESH_TOKEN'
          AND INDEX_NAME = 'IDX_MRT_REVOKED_AT'
    ) THEN
        CREATE INDEX IDX_MRT_REVOKED_AT ON ALUMNI_MOBILE_REFRESH_TOKEN (REVOKED_AT);
    END IF;
END //
DELIMITER ;

CALL _047_add_mobile_refresh_token_revoked_index_if_missing();

DROP PROCEDURE IF EXISTS _047_add_mobile_refresh_token_revoked_index_if_missing;

-- Rollback:
-- DROP INDEX IDX_MRT_REVOKED_AT ON ALUMNI_MOBILE_REFRESH_TOKEN;
