-- Migration 080: Official profile flag on feed posts.
-- WEO_BOARDBBS.OFFICIAL_PROFILE_YN = 'Y' marks a NOTICE-gate post an operator
-- published under the foundation's official profile: apps and the web show the
-- name "대일외고장학회" with the foundation emblem instead of the author.
-- REG_NAME is written as "대일외고장학회" for those posts; USR_SEQ always keeps
-- the real operator for the internal audit trail.
-- Every existing post stays 'N' (no data rewrite), so it keeps its author.
-- Target: MariaDB 10.1.38.

-- MariaDB 10.1 supports IF NOT EXISTS here, so a re-run is a no-op.
ALTER TABLE WEO_BOARDBBS ADD COLUMN IF NOT EXISTS OFFICIAL_PROFILE_YN CHAR(1) NOT NULL DEFAULT 'N';

-- Rollback:
-- ALTER TABLE WEO_BOARDBBS DROP COLUMN OFFICIAL_PROFILE_YN;
