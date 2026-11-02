-- 086: Hall of Fame wall entries and spreadsheet import batches.
-- Raw phone numbers are never stored: phone_hash is an HMAC-SHA256 keyed by a server secret.
-- Tier values (100/300/500/1000) are validated in the service layer: MariaDB 10.1 has no CHECK.
-- created_at/updated_at are set by the application (no datetime DEFAULT expressions on 10.1).
-- Rollback (empty tables only): DROP TABLE hall_of_fame_entry; DROP TABLE hall_of_fame_import_batch;
CREATE TABLE IF NOT EXISTS hall_of_fame_entry (
    id BIGINT NOT NULL AUTO_INCREMENT,
    natural_key CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    kind ENUM('alumni','companion','teacher','guest','org') NOT NULL DEFAULT 'alumni',
    cohort SMALLINT NULL,
    department VARCHAR(30) NULL,
    tier_man_won INT NULL,
    contact_name VARCHAR(100) NULL,
    phone_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    consent_public TINYINT(1) NOT NULL DEFAULT 1,
    visible TINYINT(1) NOT NULL DEFAULT 1,
    source ENUM('excel','manual') NOT NULL DEFAULT 'manual',
    source_batch_id VARCHAR(40) NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME NULL,
    PRIMARY KEY (id),
    UNIQUE KEY UK_HOF_NATURAL_KEY (natural_key),
    KEY IDX_HOF_TIER_COHORT_NAME (tier_man_won, cohort, display_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS hall_of_fame_import_batch (
    id VARCHAR(40) NOT NULL,
    file_sha256 CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    uploaded_by INT NOT NULL,
    summary_json TEXT NOT NULL,
    committed_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    KEY IDX_HOF_BATCH_FILE (file_sha256)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
