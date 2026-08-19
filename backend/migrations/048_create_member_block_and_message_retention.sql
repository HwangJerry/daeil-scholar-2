-- Migration 048: One-way member blocking and suppressed-message retention.
-- Target: MariaDB 10.1.38.
--
-- Provenance: this schema was originally defined by
-- 031_create_member_block_and_message_retention.sql, which was later archived
-- when migrations 028-039 were renumbered for the mobile-push lineage.
-- Production never applied the archived migration, so this migration restores
-- the schema under this branch's executable migration lineage.

CREATE TABLE IF NOT EXISTS ALUMNI_MEMBER_BLOCK (
    AMB_SEQ          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    BLOCKER_USR_SEQ  INT NOT NULL,
    BLOCKED_USR_SEQ  INT NOT NULL,
    CREATED_AT       DATETIME NOT NULL,
    UPDATED_AT       DATETIME NOT NULL,
    UNIQUE KEY UK_AMB_DIRECTION (BLOCKER_USR_SEQ, BLOCKED_USR_SEQ),
    INDEX IDX_AMB_BLOCKED (BLOCKED_USR_SEQ, BLOCKER_USR_SEQ)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _048_add_member_block_and_message_retention_if_missing()
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_MESSAGE'
          AND COLUMN_NAME = 'AM_CLIENT_MESSAGE_ID'
    ) THEN
        ALTER TABLE ALUMNI_MESSAGE
            ADD COLUMN AM_CLIENT_MESSAGE_ID VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL AFTER READ_DATE;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_MESSAGE'
          AND COLUMN_NAME = 'AM_VISIBLE_RECVR'
    ) THEN
        ALTER TABLE ALUMNI_MESSAGE
            ADD COLUMN AM_VISIBLE_RECVR ENUM('Y','N') NOT NULL DEFAULT 'Y' AFTER AM_CLIENT_MESSAGE_ID;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_MESSAGE'
          AND COLUMN_NAME = 'AM_SUPPRESSION_REASON'
    ) THEN
        ALTER TABLE ALUMNI_MESSAGE
            ADD COLUMN AM_SUPPRESSION_REASON VARCHAR(30) NULL AFTER AM_VISIBLE_RECVR;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_MESSAGE'
          AND COLUMN_NAME = 'PURGE_AT'
    ) THEN
        ALTER TABLE ALUMNI_MESSAGE
            ADD COLUMN PURGE_AT DATETIME NULL AFTER AM_SUPPRESSION_REASON;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_MESSAGE'
          AND INDEX_NAME = 'UK_AM_SENDER_CLIENT'
    ) THEN
        CREATE UNIQUE INDEX UK_AM_SENDER_CLIENT
            ON ALUMNI_MESSAGE (AM_SENDER_SEQ, AM_CLIENT_MESSAGE_ID);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_MESSAGE'
          AND INDEX_NAME = 'IDX_AM_RECVR_VISIBLE'
    ) THEN
        CREATE INDEX IDX_AM_RECVR_VISIBLE
            ON ALUMNI_MESSAGE (AM_RECVR_SEQ, AM_VISIBLE_RECVR, REG_DATE);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_MESSAGE'
          AND INDEX_NAME = 'IDX_AM_PURGE'
    ) THEN
        CREATE INDEX IDX_AM_PURGE ON ALUMNI_MESSAGE (PURGE_AT);
    END IF;
END //
DELIMITER ;

CALL _048_add_member_block_and_message_retention_if_missing();

DROP PROCEDURE IF EXISTS _048_add_member_block_and_message_retention_if_missing;

-- Rollback (only after reverting the application binary to a version that
-- never relied on member blocking or suppressed-message retention):
-- DROP INDEX IDX_AM_PURGE ON ALUMNI_MESSAGE;
-- DROP INDEX IDX_AM_RECVR_VISIBLE ON ALUMNI_MESSAGE;
-- DROP INDEX UK_AM_SENDER_CLIENT ON ALUMNI_MESSAGE;
-- ALTER TABLE ALUMNI_MESSAGE DROP COLUMN PURGE_AT;
-- ALTER TABLE ALUMNI_MESSAGE DROP COLUMN AM_SUPPRESSION_REASON;
-- ALTER TABLE ALUMNI_MESSAGE DROP COLUMN AM_VISIBLE_RECVR;
-- ALTER TABLE ALUMNI_MESSAGE DROP COLUMN AM_CLIENT_MESSAGE_ID;
-- DROP TABLE ALUMNI_MEMBER_BLOCK;
