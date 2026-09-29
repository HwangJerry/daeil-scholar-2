-- Migration 079: Admin-managed feed post categories.
-- ALUMNI_FEED_CATEGORY holds the categories apps show as feed tabs, ordered by
-- SORT_ORDER. FC_CODE is the stable key sent to apps as `category`; it never
-- changes on rename. Exactly one row is the default (IS_DEFAULT='Y'); the
-- application refuses to delete or hide it.
-- WEO_BOARDBBS.FEED_CATEGORY_SEQ links a NOTICE-gate post to its category.
-- NULL (every existing post) means the default category, resolved at read time.
-- Categories hold no personal data, so account erasure is unaffected.
-- Target: MariaDB 10.1.38.

CREATE TABLE IF NOT EXISTS ALUMNI_FEED_CATEGORY (
    FC_SEQ     INT NOT NULL AUTO_INCREMENT,
    FC_CODE    VARCHAR(20) NOT NULL,
    FC_NAME    VARCHAR(20) NOT NULL,
    SORT_ORDER INT NOT NULL,
    OPEN_YN    ENUM('Y','N') NOT NULL DEFAULT 'Y',
    IS_DEFAULT ENUM('Y','N') NOT NULL DEFAULT 'N',
    REG_DATE   DATETIME NULL,
    UPD_DATE   DATETIME NULL,
    PRIMARY KEY (FC_SEQ),
    UNIQUE KEY UQ_FC_CODE (FC_CODE)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed rows; UQ_FC_CODE makes a re-run a no-op.
INSERT IGNORE INTO ALUMNI_FEED_CATEGORY (FC_CODE, FC_NAME, SORT_ORDER, OPEN_YN, IS_DEFAULT, REG_DATE, UPD_DATE)
VALUES ('notice', '공지', 1, 'Y', 'Y', NOW(), NOW()),
       ('etc', '기타', 2, 'Y', 'N', NOW(), NOW());

-- MariaDB 10.1 supports IF NOT EXISTS on both, so a re-run is a no-op.
ALTER TABLE WEO_BOARDBBS ADD COLUMN IF NOT EXISTS FEED_CATEGORY_SEQ INT NULL;
CREATE INDEX IF NOT EXISTS IDX_BBS_FEED_CATEGORY ON WEO_BOARDBBS (FEED_CATEGORY_SEQ);

-- Rollback:
-- DROP INDEX IDX_BBS_FEED_CATEGORY ON WEO_BOARDBBS;
-- ALTER TABLE WEO_BOARDBBS DROP COLUMN FEED_CATEGORY_SEQ;
-- DROP TABLE ALUMNI_FEED_CATEGORY;
