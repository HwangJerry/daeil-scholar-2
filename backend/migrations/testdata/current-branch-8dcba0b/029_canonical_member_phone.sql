-- Canonicalize legacy member phone values and restore indexed lookup speed.
-- Target: MariaDB 10.1.38. Apply manually after a backup and duplicate review.
--
-- Preflight: rows returned here collapse to the same canonical phone. Resolve
-- active BBB/CCC/ZZZ duplicates before relying on phone-based signup policy.
SELECT
    REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '') AS canonical_phone,
    COUNT(*) AS member_count
FROM WEO_MEMBER
WHERE IFNULL(USR_PHONE, '') <> ''
GROUP BY REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '')
HAVING COUNT(*) > 1;

UPDATE WEO_MEMBER
SET USR_PHONE = REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '')
WHERE IFNULL(USR_PHONE, '') <> ''
  AND USR_PHONE <> REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '');

-- This numbered migration is one-time. Check information_schema.STATISTICS
-- before rerunning it manually.
CREATE INDEX IDX_WEO_MEMBER_PHONE ON WEO_MEMBER (USR_PHONE);
