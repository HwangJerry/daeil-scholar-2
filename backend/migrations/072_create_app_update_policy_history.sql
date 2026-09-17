-- Migration 072: Audit trail for app update policy changes.
-- Target: MariaDB 10.1.38.

CREATE TABLE IF NOT EXISTS app_update_policy_history (
    AUPH_SEQ    INT NOT NULL AUTO_INCREMENT,
    PLATFORM    VARCHAR(16) NOT NULL,
    BEFORE_JSON TEXT NOT NULL,
    AFTER_JSON  TEXT NOT NULL,
    CHANGED_BY  INT NULL,
    CHANGED_AT  DATETIME NOT NULL,
    PRIMARY KEY (AUPH_SEQ),
    INDEX IDX_AUPH_PLATFORM_CHANGED (PLATFORM, CHANGED_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Rollback:
-- DROP TABLE app_update_policy_history;
