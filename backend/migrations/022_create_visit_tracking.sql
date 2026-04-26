-- =============================================================================
-- 022: Create visit tracking tables for DAU/MAU analytics
-- Target: MariaDB 10.1.38
-- Safe to re-run: uses IF NOT EXISTS.
-- =============================================================================

-- WEO_VISIT_DAILY — one row per (date, visitor). Upserted by the beacon endpoint.
-- VD_USR_SEQ = 0 for anonymous visitors; upgraded on same-day login via UPSERT.
CREATE TABLE IF NOT EXISTS WEO_VISIT_DAILY (
    VD_DATE        DATE         NOT NULL,
    VD_VISITOR_ID  CHAR(36)     NOT NULL,
    VD_USR_SEQ     INT          NOT NULL DEFAULT 0,
    VD_FIRST_TS    DATETIME     NOT NULL,
    VD_LAST_TS     DATETIME     NOT NULL,
    VD_HITS        INT UNSIGNED NOT NULL DEFAULT 1,
    VD_UA_HASH     CHAR(16)     DEFAULT NULL,
    VD_IP_HASH     CHAR(16)     DEFAULT NULL,
    PRIMARY KEY (VD_DATE, VD_VISITOR_ID),
    KEY IX_VD_DATE_USR (VD_DATE, VD_USR_SEQ)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- WEO_VISIT_SUMMARY — pre-aggregated per-day counters written by the daily job.
-- VS_MAU_TOTAL is the 30-day rolling distinct visitor count ending on VS_DATE.
CREATE TABLE IF NOT EXISTS WEO_VISIT_SUMMARY (
    VS_DATE        DATE          NOT NULL PRIMARY KEY,
    VS_DAU_TOTAL   INT UNSIGNED  NOT NULL DEFAULT 0,
    VS_DAU_MEMBER  INT UNSIGNED  NOT NULL DEFAULT 0,
    VS_DAU_ANON    INT UNSIGNED  NOT NULL DEFAULT 0,
    VS_MAU_TOTAL   INT UNSIGNED  NOT NULL DEFAULT 0,
    VS_PAGEVIEWS   INT UNSIGNED  NOT NULL DEFAULT 0,
    REG_DATE       DATETIME      NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
