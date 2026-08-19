-- 034_social_revocation_outbox_claim.sql — Atomic row-claiming for the social
-- revocation outbox worker (internal/job/social_revocation_worker.go).
--
-- Without this, ListDueSocialRevocations was a plain unlocked SELECT: two
-- concurrent worker processes (e.g. an overlapping restart, or a
-- misconfigured second instance) could both fetch and process the same
-- pending row, each calling the upstream Kakao/Apple revoke API for the same
-- credential. CLAIM_TOKEN lets a worker atomically "check out" a batch of due
-- rows via a single UPDATE before processing them (see
-- ClaimDueSocialRevocations), so a second worker's claim UPDATE cannot match
-- rows already claimed and not yet stale.
-- Target: MariaDB 10.1.38. This numbered migration is one-time; the column
-- and index guards below make it safe to rerun manually if needed.

DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _add_social_revocation_claim_token_if_missing()
BEGIN
    SET @col_exists = 0;
    SELECT COUNT(*) INTO @col_exists
      FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'ALUMNI_SOCIAL_REVOCATION_OUTBOX'
       AND COLUMN_NAME = 'CLAIM_TOKEN';
    IF @col_exists = 0 THEN
        ALTER TABLE ALUMNI_SOCIAL_REVOCATION_OUTBOX
            ADD COLUMN CLAIM_TOKEN VARCHAR(64) NULL AFTER STATUS;
    END IF;

    SET @idx_exists = 0;
    SELECT COUNT(DISTINCT INDEX_NAME) INTO @idx_exists
      FROM information_schema.STATISTICS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'ALUMNI_SOCIAL_REVOCATION_OUTBOX'
       AND INDEX_NAME = 'IDX_SRO_CLAIM';
    IF @idx_exists = 0 THEN
        CREATE INDEX IDX_SRO_CLAIM ON ALUMNI_SOCIAL_REVOCATION_OUTBOX (CLAIM_TOKEN);
    END IF;
END //
DELIMITER ;

CALL _add_social_revocation_claim_token_if_missing();

DROP PROCEDURE IF EXISTS _add_social_revocation_claim_token_if_missing;
