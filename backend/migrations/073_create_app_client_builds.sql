-- Migration 073: Build numbers observed from mobile API requests.
-- Feeds the admin build picker so operators never type a raw build number.
-- Aggregate only: no user, session, device, or IP identifiers are stored.
-- Target: MariaDB 10.1.38.

CREATE TABLE IF NOT EXISTS app_client_builds (
    PLATFORM      VARCHAR(16) NOT NULL,
    BUILD         BIGINT NOT NULL,
    VERSION_NAME  VARCHAR(64) NOT NULL DEFAULT '',
    FIRST_SEEN_AT DATETIME NOT NULL,
    LAST_SEEN_AT  DATETIME NOT NULL,
    PRIMARY KEY (PLATFORM, BUILD),
    INDEX IDX_ACB_PLATFORM_LAST_SEEN (PLATFORM, LAST_SEEN_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Rollback:
-- DROP TABLE app_client_builds;
