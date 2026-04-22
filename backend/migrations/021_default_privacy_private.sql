-- =============================================================================
-- 021: Flip default privacy to private ('N') for phone/email visibility
--      and backfill existing members to private.
-- Target: MariaDB 10.1.38
-- Safe to re-run: UPDATE is idempotent; ALTER re-sets the same default.
-- =============================================================================

-- Backfill all existing rows to 'N' (private).
UPDATE WEO_MEMBER SET USR_PHONE_PUBLIC = 'N' WHERE USR_PHONE_PUBLIC <> 'N';
UPDATE WEO_MEMBER SET USR_EMAIL_PUBLIC = 'N' WHERE USR_EMAIL_PUBLIC <> 'N';

-- Flip the column default so future INSERTs without an explicit value are private.
ALTER TABLE WEO_MEMBER
    MODIFY COLUMN USR_PHONE_PUBLIC ENUM('Y','N') NOT NULL DEFAULT 'N';
ALTER TABLE WEO_MEMBER
    MODIFY COLUMN USR_EMAIL_PUBLIC ENUM('Y','N') NOT NULL DEFAULT 'N';
