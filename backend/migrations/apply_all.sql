-- =============================================================================
-- apply_all.sql — Consolidated migration script (001–023)
-- Target: MariaDB 10.1.38
-- Safe to re-run: uses IF NOT EXISTS / procedure-based column checks
-- =============================================================================

-- Helper procedure: Add column only if it doesn't already exist
DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _add_column_if_not_exists(
    IN p_table VARCHAR(64),
    IN p_column VARCHAR(64),
    IN p_definition TEXT
)
BEGIN
    SET @col_exists = 0;
    SELECT COUNT(*) INTO @col_exists
      FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = p_table
       AND COLUMN_NAME = p_column;
    IF @col_exists = 0 THEN
        SET @ddl = CONCAT('ALTER TABLE ', p_table, ' ADD COLUMN ', p_column, ' ', p_definition);
        PREPARE stmt FROM @ddl;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END //

-- Helper procedure: Create index only if it doesn't already exist
CREATE PROCEDURE IF NOT EXISTS _add_index_if_not_exists(
    IN p_table VARCHAR(64),
    IN p_index VARCHAR(64),
    IN p_columns TEXT
)
BEGIN
    SET @idx_exists = 0;
    SELECT COUNT(*) INTO @idx_exists
      FROM information_schema.STATISTICS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = p_table
       AND INDEX_NAME = p_index;
    IF @idx_exists = 0 THEN
        SET @ddl = CONCAT('CREATE INDEX ', p_index, ' ON ', p_table, ' (', p_columns, ')');
        PREPARE stmt FROM @ddl;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END //
DELIMITER ;


-- =============================================================================
-- 001: Alter existing tables for social feed features
-- =============================================================================
CALL _add_column_if_not_exists('WEO_BOARDBBS', 'THUMBNAIL_URL', "VARCHAR(500) NULL COMMENT '대표 이미지 URL' AFTER FILES");
CALL _add_column_if_not_exists('WEO_BOARDBBS', 'SUMMARY', "VARCHAR(200) NULL COMMENT '본문 요약' AFTER THUMBNAIL_URL");
CALL _add_column_if_not_exists('WEO_BOARDBBS', 'IS_PINNED', "ENUM('Y','N') DEFAULT 'N' COMMENT '상단 고정 여부' AFTER SUMMARY");

CALL _add_column_if_not_exists('MAIN_AD', 'AD_TIER', "ENUM('PREMIUM','GOLD','NORMAL') DEFAULT 'NORMAL' COMMENT '광고 등급' AFTER MA_TYPE");
CALL _add_column_if_not_exists('MAIN_AD', 'AD_TITLE_LABEL', "VARCHAR(50) DEFAULT '추천 동문 소식' COMMENT '광고 카드 타이틀 라벨' AFTER AD_TIER");

CALL _add_index_if_not_exists('FUNDAMENTAL_MEMBER', 'INDX_COMPANY', 'FM_COMPANY');
CALL _add_index_if_not_exists('FUNDAMENTAL_MEMBER', 'INDX_POSITION', 'FM_POSITION');


-- =============================================================================
-- 002: Create new InnoDB tables
-- =============================================================================
CREATE TABLE IF NOT EXISTS DONATION_SNAPSHOT (
    DS_SEQ        INT AUTO_INCREMENT PRIMARY KEY,
    DS_DATE       DATE NOT NULL,
    DS_TOTAL      BIGINT DEFAULT 0 NOT NULL,
    DS_MANUAL_ADJ BIGINT DEFAULT 0 NOT NULL,
    DS_DONOR_CNT  INT DEFAULT 0 NOT NULL,
    DS_GOAL       BIGINT DEFAULT 0 NOT NULL,
    REG_DATE      DATETIME NULL,
    UNIQUE KEY UK_DATE (DS_DATE)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS DONATION_CONFIG (
    DC_SEQ        INT AUTO_INCREMENT PRIMARY KEY,
    DC_GOAL       BIGINT DEFAULT 0 NOT NULL,
    DC_MANUAL_ADJ BIGINT DEFAULT 0 NOT NULL,
    DC_NOTE       VARCHAR(200) NULL,
    IS_ACTIVE     ENUM('Y','N') DEFAULT 'Y',
    REG_DATE      DATETIME NULL,
    REG_OPER      INT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS USER_SESSION (
    SESSION_ID    VARCHAR(64) NOT NULL PRIMARY KEY,
    USR_SEQ       INT NOT NULL,
    PROVIDER      ENUM('KAKAO','DIRECT') DEFAULT 'DIRECT',
    EXPIRES_AT    DATETIME NOT NULL,
    CREATED_AT    DATETIME NOT NULL,
    INDEX IDX_USR (USR_SEQ),
    INDEX IDX_EXPIRES (EXPIRES_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS WEO_AD_LOG (
    AL_SEQ     INT AUTO_INCREMENT PRIMARY KEY,
    MA_SEQ     INT NOT NULL,
    USR_SEQ    INT NULL,
    AL_TYPE    ENUM('VIEW','CLICK') NOT NULL,
    AL_DATE    DATETIME NOT NULL,
    AL_IPADDR  VARCHAR(45) NULL,
    INDEX IDX_MA_SEQ (MA_SEQ),
    INDEX IDX_DATE (AL_DATE)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- =============================================================================
-- 003: Seed initial donation config (skip if already seeded)
-- =============================================================================
INSERT INTO DONATION_CONFIG (DC_GOAL, DC_MANUAL_ADJ, DC_NOTE, IS_ACTIVE, REG_DATE)
SELECT 200000000, 0, '초기 설정', 'Y', NOW()
  FROM DUAL
 WHERE NOT EXISTS (SELECT 1 FROM DONATION_CONFIG WHERE IS_ACTIVE = 'Y');


-- =============================================================================
-- 004: Add Markdown content columns to WEO_BOARDBBS
-- =============================================================================
CALL _add_column_if_not_exists('WEO_BOARDBBS', 'CONTENTS_MD', "MEDIUMTEXT NULL COMMENT '원본 Markdown 텍스트 (편집용)' AFTER CONTENTS");
CALL _add_column_if_not_exists('WEO_BOARDBBS', 'CONTENT_FORMAT', "ENUM('LEGACY','MARKDOWN') DEFAULT 'LEGACY' COMMENT '콘텐츠 포맷' AFTER CONTENTS_MD");


-- =============================================================================
-- 005: Add ad image column to MAIN_AD
-- =============================================================================
CALL _add_column_if_not_exists('MAIN_AD', 'MA_IMG', "VARCHAR(500) NULL COMMENT '광고 배너 이미지 URL' AFTER MA_URL");


-- =============================================================================
-- 006: Add USR_PHOTO column to WEO_MEMBER
-- =============================================================================
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_PHOTO', 'VARCHAR(500) DEFAULT NULL');


-- =============================================================================
-- 007: Create WEO_MEMBER_SOCIAL table for OAuth account linking
-- =============================================================================
CREATE TABLE IF NOT EXISTS WEO_MEMBER_SOCIAL (
    NMS_SEQ   INT AUTO_INCREMENT PRIMARY KEY,
    USR_SEQ   INT NOT NULL,
    NMS_GATE  VARCHAR(10) NOT NULL,
    NMS_ID    VARCHAR(100) NOT NULL,
    NMS_NAME  VARCHAR(100) NULL,
    REG_DATE  DATETIME NULL,
    INDEX IDX_USR_SEQ (USR_SEQ),
    UNIQUE KEY UK_GATE_ID (NMS_GATE, NMS_ID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- =============================================================================
-- 008: Alumni profile extensions — job categories, user tags, business info
-- =============================================================================
CREATE TABLE IF NOT EXISTS ALUMNI_JOB_CATEGORY (
    AJC_SEQ   INT AUTO_INCREMENT PRIMARY KEY,
    AJC_NAME  VARCHAR(50) NOT NULL,
    AJC_COLOR VARCHAR(7) DEFAULT '#4F46E5',
    AJC_INDX  INT DEFAULT 0,
    OPEN_YN   ENUM('Y','N') DEFAULT 'Y',
    REG_DATE  DATETIME NULL,
    UNIQUE KEY UK_NAME (AJC_NAME)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_USER_TAG (
    AUT_SEQ  INT AUTO_INCREMENT PRIMARY KEY,
    USR_SEQ  INT NOT NULL,
    AUT_TAG  VARCHAR(30) NOT NULL,
    AUT_INDX INT DEFAULT 0,
    REG_DATE DATETIME NULL,
    INDEX IDX_USR_SEQ (USR_SEQ),
    UNIQUE KEY UK_USR_TAG (USR_SEQ, AUT_TAG)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_BIZ_NAME', 'VARCHAR(100) NULL AFTER USR_PHOTO');
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_BIZ_DESC', 'VARCHAR(200) NULL AFTER USR_BIZ_NAME');
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_BIZ_ADDR', 'VARCHAR(200) NULL AFTER USR_BIZ_DESC');
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_JOB_CAT', 'INT NULL AFTER USR_BIZ_ADDR');

-- Seed job categories (skip duplicates via INSERT IGNORE + unique key)
INSERT IGNORE INTO ALUMNI_JOB_CATEGORY (AJC_NAME, AJC_COLOR, AJC_INDX, OPEN_YN, REG_DATE) VALUES
    ('부동산',      '#DC2626', 1,  'Y', NOW()),
    ('IT/테크',     '#2563EB', 2,  'Y', NOW()),
    ('의료/건강',   '#059669', 3,  'Y', NOW()),
    ('교육',        '#D97706', 4,  'Y', NOW()),
    ('금융/보험',   '#7C3AED', 5,  'Y', NOW()),
    ('법률',        '#4338CA', 6,  'Y', NOW()),
    ('여행/관광',   '#0891B2', 7,  'Y', NOW()),
    ('스포츠/헬스', '#EA580C', 8,  'Y', NOW()),
    ('요식업',      '#CA8A04', 9,  'Y', NOW()),
    ('기타',        '#6B7280', 10, 'Y', NOW());


-- =============================================================================
-- 009: Create alumni message table
-- =============================================================================
CREATE TABLE IF NOT EXISTS ALUMNI_MESSAGE (
    AM_SEQ        INT AUTO_INCREMENT PRIMARY KEY,
    AM_SENDER_SEQ INT NOT NULL,
    AM_RECVR_SEQ  INT NOT NULL,
    AM_CONTENT    TEXT NOT NULL,
    AM_READ_YN    ENUM('Y','N') DEFAULT 'N',
    AM_DEL_SENDER ENUM('Y','N') DEFAULT 'N',
    AM_DEL_RECVR  ENUM('Y','N') DEFAULT 'N',
    REG_DATE      DATETIME NULL,
    READ_DATE     DATETIME NULL,
    INDEX IDX_SENDER (AM_SENDER_SEQ, REG_DATE),
    INDEX IDX_RECVR (AM_RECVR_SEQ, AM_READ_YN, REG_DATE)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- =============================================================================
-- 010: Create subscription table for recurring donations
-- =============================================================================
CREATE TABLE IF NOT EXISTS SUBSCRIPTION (
    SUB_SEQ    INT AUTO_INCREMENT PRIMARY KEY,
    USR_SEQ    INT NOT NULL,
    AMOUNT     INT NOT NULL,
    PAY_TYPE   VARCHAR(10) NOT NULL DEFAULT 'CARD',
    STATUS     VARCHAR(20) NOT NULL DEFAULT 'active',
    START_DATE DATETIME NOT NULL,
    NEXT_BILL  DATETIME NOT NULL,
    REG_DATE   DATETIME NOT NULL,
    EDT_DATE   DATETIME,
    INDEX idx_usr_seq (USR_SEQ)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- =============================================================================
-- 011: Create WEO_AD_LIKE table
-- =============================================================================
CREATE TABLE IF NOT EXISTS WEO_AD_LIKE (
    AL_SEQ   INT AUTO_INCREMENT PRIMARY KEY,
    MA_SEQ   INT NOT NULL,
    USR_SEQ  INT NOT NULL,
    OPEN_YN  ENUM('Y','N') NOT NULL DEFAULT 'Y',
    REG_DATE DATETIME NOT NULL,
    INDEX IDX_MA_USR (MA_SEQ, USR_SEQ)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- =============================================================================
-- 012: Create WEO_AD_COMMENT table
-- =============================================================================
CREATE TABLE IF NOT EXISTS WEO_AD_COMMENT (
    AC_SEQ   INT AUTO_INCREMENT PRIMARY KEY,
    MA_SEQ   INT NOT NULL,
    USR_SEQ  INT NOT NULL,
    NICKNAME VARCHAR(100) NOT NULL,
    CONTENTS TEXT NOT NULL,
    OPEN_YN  ENUM('Y','N') NOT NULL DEFAULT 'Y',
    REG_DATE DATETIME NOT NULL,
    INDEX IDX_MA_SEQ (MA_SEQ)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- =============================================================================
-- 013: Performance indexes
-- =============================================================================
CALL _add_index_if_not_exists('WEO_BOARDBBS', 'IDX_BBS_FEED', 'GATE, OPEN_YN, SEQ');
CALL _add_index_if_not_exists('WEO_BOARDLIKE', 'IDX_BBS_LIKE', 'BBS_SEQ, OPEN_YN');
CALL _add_index_if_not_exists('WEO_BOARDCOMAND', 'IDX_BCOM_JOIN', 'JOIN_SEQ, BC_TYPE, OPEN_YN');
CALL _add_index_if_not_exists('FUNDAMENTAL_MEMBER', 'IDX_FM_FN', 'FM_FN');
CALL _add_index_if_not_exists('WEO_MEMBER', 'IDX_USR_STATUS', 'USR_STATUS');
CALL _add_index_if_not_exists('WEO_MEMBER', 'IDX_USR_REG_DATE', 'REG_DATE');


-- =============================================================================
-- 014: Add USR_DEPT to WEO_MEMBER
-- =============================================================================
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_DEPT', 'VARCHAR(100) NULL AFTER USR_FN');


-- =============================================================================
-- 015: Add profile privacy toggles and business card
-- =============================================================================
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_PHONE_PUBLIC', "ENUM('Y','N') NOT NULL DEFAULT 'Y'");
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_EMAIL_PUBLIC', "ENUM('Y','N') NOT NULL DEFAULT 'Y'");
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_BIZ_CARD', 'VARCHAR(500) NULL');


-- =============================================================================
-- 016: Create password reset table
-- =============================================================================
CREATE TABLE IF NOT EXISTS ALUMNI_PASSWORD_RESET (
    APR_SEQ      INT AUTO_INCREMENT PRIMARY KEY,
    USR_SEQ      INT NOT NULL,
    APR_TOKEN    VARCHAR(64) NOT NULL,
    APR_USED_YN  ENUM('Y','N') DEFAULT 'N',
    EXPIRES_AT   DATETIME NOT NULL,
    REG_DATE     DATETIME NULL,
    UNIQUE INDEX IDX_TOKEN (APR_TOKEN),
    INDEX IDX_USR (USR_SEQ)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- =============================================================================
-- 017: Create notification table
-- =============================================================================
CREATE TABLE IF NOT EXISTS ALUMNI_NOTIFICATION (
    AN_SEQ       INT AUTO_INCREMENT PRIMARY KEY,
    USR_SEQ      INT NOT NULL,
    AN_TYPE      VARCHAR(30) NOT NULL,
    AN_TITLE     VARCHAR(200) NOT NULL,
    AN_BODY      VARCHAR(500) NULL,
    AN_REF_SEQ   INT NULL,
    AN_READ_YN   ENUM('Y','N') DEFAULT 'N',
    REG_DATE     DATETIME NULL,
    INDEX IDX_USR_READ (USR_SEQ, AN_READ_YN, REG_DATE),
    INDEX IDX_USR_DATE (USR_SEQ, REG_DATE)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- =============================================================================
-- 021: Flip privacy default to private ('N') and backfill existing members
-- =============================================================================
UPDATE WEO_MEMBER SET USR_PHONE_PUBLIC = 'N' WHERE USR_PHONE_PUBLIC <> 'N';
UPDATE WEO_MEMBER SET USR_EMAIL_PUBLIC = 'N' WHERE USR_EMAIL_PUBLIC <> 'N';
ALTER TABLE WEO_MEMBER
    MODIFY COLUMN USR_PHONE_PUBLIC ENUM('Y','N') NOT NULL DEFAULT 'N';
ALTER TABLE WEO_MEMBER
    MODIFY COLUMN USR_EMAIL_PUBLIC ENUM('Y','N') NOT NULL DEFAULT 'N';


-- =============================================================================
-- 023: Visit tracking tables for DAU/MAU
-- =============================================================================
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

CREATE TABLE IF NOT EXISTS WEO_VISIT_SUMMARY (
    VS_DATE        DATE          NOT NULL PRIMARY KEY,
    VS_DAU_TOTAL   INT UNSIGNED  NOT NULL DEFAULT 0,
    VS_DAU_MEMBER  INT UNSIGNED  NOT NULL DEFAULT 0,
    VS_DAU_ANON    INT UNSIGNED  NOT NULL DEFAULT 0,
    VS_MAU_TOTAL   INT UNSIGNED  NOT NULL DEFAULT 0,
    VS_PAGEVIEWS   INT UNSIGNED  NOT NULL DEFAULT 0,
    REG_DATE       DATETIME      NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- =============================================================================
-- Cleanup: Drop helper procedures
-- =============================================================================
DROP PROCEDURE IF EXISTS _add_column_if_not_exists;
DROP PROCEDURE IF EXISTS _add_index_if_not_exists;


-- =============================================================================
-- Verification queries
-- =============================================================================
SELECT '=== VERIFICATION ===' AS status;

-- 001: Feed columns
SELECT 'WEO_BOARDBBS.IS_PINNED' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_BOARDBBS' AND COLUMN_NAME='IS_PINNED';
SELECT 'WEO_BOARDBBS.SUMMARY' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_BOARDBBS' AND COLUMN_NAME='SUMMARY';
SELECT 'WEO_BOARDBBS.THUMBNAIL_URL' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_BOARDBBS' AND COLUMN_NAME='THUMBNAIL_URL';

-- 002: New tables
SELECT 'DONATION_SNAPSHOT' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='DONATION_SNAPSHOT';
SELECT 'USER_SESSION' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='USER_SESSION';

-- 006: USR_PHOTO
SELECT 'WEO_MEMBER.USR_PHOTO' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER' AND COLUMN_NAME='USR_PHOTO';

-- 007: Social table
SELECT 'WEO_MEMBER_SOCIAL' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL';

-- 008: Job categories
SELECT 'ALUMNI_JOB_CATEGORY count' AS chk, COUNT(*) AS found FROM ALUMNI_JOB_CATEGORY;

-- 014: USR_DEPT
SELECT 'WEO_MEMBER.USR_DEPT' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER' AND COLUMN_NAME='USR_DEPT';

-- 015: Privacy columns
SELECT 'WEO_MEMBER.USR_PHONE_PUBLIC' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER' AND COLUMN_NAME='USR_PHONE_PUBLIC';

-- 016: Password reset table
SELECT 'ALUMNI_PASSWORD_RESET' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_PASSWORD_RESET';

-- 017: Notification table
SELECT 'ALUMNI_NOTIFICATION' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_NOTIFICATION';

-- 023: Visit tracking tables
SELECT 'WEO_VISIT_DAILY' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_VISIT_DAILY';
SELECT 'WEO_VISIT_SUMMARY' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_VISIT_SUMMARY';

-- 021: Privacy default flipped to 'N'
SELECT 'USR_PHONE_PUBLIC default=N' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER' AND COLUMN_NAME='USR_PHONE_PUBLIC' AND COLUMN_DEFAULT='N';
SELECT 'USR_EMAIL_PUBLIC default=N' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER' AND COLUMN_NAME='USR_EMAIL_PUBLIC' AND COLUMN_DEFAULT='N';

SELECT '=== ALL MIGRATIONS APPLIED ===' AS status;
