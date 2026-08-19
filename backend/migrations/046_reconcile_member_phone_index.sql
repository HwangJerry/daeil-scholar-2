-- Migration 046: Canonicalize legacy WEO_MEMBER.USR_PHONE values (strip hyphens
-- and spaces) and restore an indexed lookup path for phone-based member search.
-- Target: MariaDB 10.1.38.
--
-- Provenance: mirrors archive/main-pre-integration-20260819's
-- 033_canonical_member_phone.sql, which was manually applied directly to
-- production on 2026-08-19 (see migration 045's header for context). This
-- migration is idempotent and safe to rerun; the UPDATE only touches rows
-- whose USR_PHONE isn't already canonical, and the index creation is guarded.
--
-- Preflight note (informational only, does not block this migration - this
-- index is non-unique): as of 2026-08-19, 12 canonical-phone groups collapse
-- multiple WEO_MEMBER rows to the same normalized phone number, mostly
-- pre-existing duplicate/placeholder legacy data. Resolve those manually
-- (see USR_STATUS='AAA' withdrawal, not deletion, for confirmed duplicates)
-- before relying on phone uniqueness in any future signup/lookup policy:
--   SELECT REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '') AS canonical_phone,
--          COUNT(*) AS member_count
--   FROM WEO_MEMBER
--   WHERE IFNULL(USR_PHONE, '') <> ''
--   GROUP BY REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '')
--   HAVING COUNT(*) > 1;

UPDATE WEO_MEMBER
SET USR_PHONE = REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '')
WHERE IFNULL(USR_PHONE, '') <> ''
  AND USR_PHONE <> REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '');

DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _046_add_member_phone_index_if_missing()
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'WEO_MEMBER'
          AND INDEX_NAME = 'IDX_WEO_MEMBER_PHONE'
    ) THEN
        CREATE INDEX IDX_WEO_MEMBER_PHONE ON WEO_MEMBER (USR_PHONE);
    END IF;
END //
DELIMITER ;

CALL _046_add_member_phone_index_if_missing();

DROP PROCEDURE IF EXISTS _046_add_member_phone_index_if_missing;

-- Rollback (only after reverting the application binary to a version that
-- never relied on this index; the USR_PHONE canonicalization itself is not
-- reversible - the original hyphenated/spaced formatting is not recoverable
-- from the canonical value):
-- DROP INDEX IDX_WEO_MEMBER_PHONE ON WEO_MEMBER;
