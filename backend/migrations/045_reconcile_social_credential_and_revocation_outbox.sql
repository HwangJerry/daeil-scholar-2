-- Migration 045: Apple Sign In replay/nonce guards, encrypted social provider
-- credential storage, and the atomic-claim revocation outbox drained by
-- internal/job/social_revocation_worker.go.
-- Target: MariaDB 10.1.38.
--
-- Provenance: this reconciles schema that was designed and manually applied
-- directly to production on 2026-08-19 (see archive/main-pre-integration-20260819's
-- 032_social_account_management.sql and 034_social_revocation_outbox_claim.sql)
-- while investigating a branch-lineage mismatch, before this repository's real
-- 032-044 numbers (a different, unrelated push/canonical-identity lineage) were
-- discovered to already be production-authoritative. WEO_MEMBER_SOCIAL's
-- NMS_EMAIL/NMS_STATUS/NMS_EMAIL_ENABLED/UK_USR_PROVIDER are NOT touched here -
-- migration 037 already covers them. ALUMNI_SOCIAL_CREDENTIAL and
-- ALUMNI_SOCIAL_REVOCATION_OUTBOX are not new/duplicate tables: this branch's
-- own backend/internal/repository/auth_repo.go already reads/writes them
-- directly (UpsertSocialCredential, GetSocialCredential, EnqueueSocialRevocation,
-- etc.), but no migration in the 001-044 chain actually creates them - this
-- migration closes that gap.
-- This migration is idempotent (safe to rerun) and safe to apply on top of the
-- tables as they already exist in production from the 2026-08-19 manual apply.

CREATE TABLE IF NOT EXISTS ALUMNI_APPLE_NONCE_CHALLENGE (
    CHALLENGE_ID VARCHAR(64) NOT NULL PRIMARY KEY,
    NONCE_HASH   CHAR(64) NOT NULL,
    EXPIRES_AT   DATETIME NOT NULL,
    CONSUMED_AT  DATETIME NULL,
    CREATED_AT   DATETIME NOT NULL,
    INDEX IDX_ANC_EXPIRES (EXPIRES_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_APPLE_CODE_REPLAY (
    CODE_HASH  CHAR(64) NOT NULL PRIMARY KEY,
    CREATED_AT DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_SOCIAL_CREDENTIAL (
    USR_SEQ              INT NOT NULL,
    PROVIDER             VARCHAR(10) NOT NULL,
    ENCRYPTED_CREDENTIAL TEXT NOT NULL,
    UPDATED_AT           DATETIME NOT NULL,
    PRIMARY KEY (USR_SEQ, PROVIDER)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_SOCIAL_REVOCATION_OUTBOX (
    OUTBOX_ID       BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    USR_SEQ         INT NOT NULL,
    PROVIDER        VARCHAR(10) NOT NULL,
    ACTION          VARCHAR(30) NOT NULL,
    STATUS          VARCHAR(20) NOT NULL,
    ATTEMPT_COUNT   INT NOT NULL DEFAULT 0,
    NEXT_ATTEMPT_AT DATETIME NOT NULL,
    LAST_ERROR      VARCHAR(500) NULL,
    CREATED_AT      DATETIME NOT NULL,
    UPDATED_AT      DATETIME NOT NULL,
    INDEX IDX_SRO_DUE (STATUS, NEXT_ATTEMPT_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- CLAIM_TOKEN lets the ported social_revocation_worker atomically "check out"
-- a batch of due rows via a single UPDATE before processing them, so two
-- concurrent worker processes (an overlapping restart, a misconfigured second
-- instance) cannot both claim and process the same row. Guarded because the
-- base table may already exist (from the 2026-08-19 manual apply, or a prior
-- run of this migration) without this column.
DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _045_add_social_revocation_claim_token_if_missing()
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_SOCIAL_REVOCATION_OUTBOX'
          AND COLUMN_NAME = 'CLAIM_TOKEN'
    ) THEN
        ALTER TABLE ALUMNI_SOCIAL_REVOCATION_OUTBOX
            ADD COLUMN CLAIM_TOKEN VARCHAR(64) NULL AFTER STATUS;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'ALUMNI_SOCIAL_REVOCATION_OUTBOX'
          AND INDEX_NAME = 'IDX_SRO_CLAIM'
    ) THEN
        CREATE INDEX IDX_SRO_CLAIM ON ALUMNI_SOCIAL_REVOCATION_OUTBOX (CLAIM_TOKEN);
    END IF;
END //
DELIMITER ;

CALL _045_add_social_revocation_claim_token_if_missing();

DROP PROCEDURE IF EXISTS _045_add_social_revocation_claim_token_if_missing;

-- Rollback (only after reverting the application binary to a version that
-- never wrote these tables/column):
-- DROP INDEX IDX_SRO_CLAIM ON ALUMNI_SOCIAL_REVOCATION_OUTBOX;
-- ALTER TABLE ALUMNI_SOCIAL_REVOCATION_OUTBOX DROP COLUMN CLAIM_TOKEN;
-- DROP TABLE ALUMNI_SOCIAL_REVOCATION_OUTBOX;
-- DROP TABLE ALUMNI_SOCIAL_CREDENTIAL;
-- DROP TABLE ALUMNI_APPLE_CODE_REPLAY;
-- DROP TABLE ALUMNI_APPLE_NONCE_CHALLENGE;
