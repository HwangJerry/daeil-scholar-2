-- =============================================================================
-- apply_all.sql — Consolidated migration script (001–034)
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
    SET @idx_columns = NULL;
    SET @idx_non_unique = NULL;
    SELECT COUNT(*),
           GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ','),
           MIN(NON_UNIQUE)
      INTO @idx_exists, @idx_columns, @idx_non_unique
      FROM information_schema.STATISTICS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = p_table
       AND INDEX_NAME = p_index;
    IF @idx_exists = 0 THEN
        SET @ddl = CONCAT('CREATE INDEX ', p_index, ' ON ', p_table, ' (', p_columns, ')');
        PREPARE stmt FROM @ddl;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    ELSEIF REPLACE(UPPER(@idx_columns), ' ', '') <> REPLACE(UPPER(p_columns), ' ', '')
        OR @idx_non_unique <> 1 THEN
        SELECT p_table AS TABLE_NAME, p_index AS INDEX_NAME,
               @idx_columns AS ACTUAL_COLUMNS, p_columns AS EXPECTED_COLUMNS,
               @idx_non_unique AS ACTUAL_NON_UNIQUE, 1 AS EXPECTED_NON_UNIQUE;
        -- Reusing the conflicting name deliberately raises error 1061.
        SET @ddl = CONCAT('CREATE INDEX ', p_index, ' ON ', p_table, ' (', p_columns, ')');
        PREPARE stmt FROM @ddl;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END //

CREATE PROCEDURE IF NOT EXISTS _add_unique_index_if_not_exists(
    IN p_table VARCHAR(64),
    IN p_index VARCHAR(64),
    IN p_columns TEXT
)
BEGIN
    SET @idx_exists = 0;
    SET @idx_columns = NULL;
    SET @idx_non_unique = NULL;
    SELECT COUNT(*),
           GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ','),
           MIN(NON_UNIQUE)
      INTO @idx_exists, @idx_columns, @idx_non_unique
      FROM information_schema.STATISTICS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = p_table
       AND INDEX_NAME = p_index;
    IF @idx_exists = 0 THEN
        SET @ddl = CONCAT('CREATE UNIQUE INDEX ', p_index, ' ON ', p_table, ' (', p_columns, ')');
        PREPARE stmt FROM @ddl;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    ELSEIF REPLACE(UPPER(@idx_columns), ' ', '') <> REPLACE(UPPER(p_columns), ' ', '')
        OR @idx_non_unique <> 0 THEN
        SELECT p_table AS TABLE_NAME, p_index AS INDEX_NAME,
               @idx_columns AS ACTUAL_COLUMNS, p_columns AS EXPECTED_COLUMNS,
               @idx_non_unique AS ACTUAL_NON_UNIQUE, 0 AS EXPECTED_NON_UNIQUE;
        -- Reusing the conflicting name deliberately raises error 1061.
        SET @ddl = CONCAT('CREATE UNIQUE INDEX ', p_index, ' ON ', p_table, ' (', p_columns, ')');
        PREPARE stmt FROM @ddl;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END //
DELIMITER ;

CREATE TABLE IF NOT EXISTS ALUMNI_SCHEMA_MIGRATION (
    MIGRATION_ID VARCHAR(100) CHARACTER SET ascii COLLATE ascii_bin NOT NULL PRIMARY KEY,
    APPLIED_AT DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- =============================================================================
-- 001: Alter existing tables for social feed features
-- =============================================================================
CALL _add_column_if_not_exists('WEO_BOARDBBS', 'THUMBNAIL_URL', 'VARCHAR(500) NULL COMMENT ''대표 이미지 URL'' AFTER FILES');
CALL _add_column_if_not_exists('WEO_BOARDBBS', 'SUMMARY', 'VARCHAR(200) NULL COMMENT ''본문 요약'' AFTER THUMBNAIL_URL');
CALL _add_column_if_not_exists('WEO_BOARDBBS', 'IS_PINNED', 'ENUM(''Y'',''N'') DEFAULT ''N'' COMMENT ''상단 고정 여부'' AFTER SUMMARY');

CALL _add_column_if_not_exists('MAIN_AD', 'AD_TIER', 'ENUM(''PREMIUM'',''GOLD'',''NORMAL'') DEFAULT ''NORMAL'' COMMENT ''광고 등급'' AFTER MA_TYPE');
CALL _add_column_if_not_exists('MAIN_AD', 'AD_TITLE_LABEL', 'VARCHAR(50) DEFAULT ''추천 동문 소식'' COMMENT ''광고 카드 타이틀 라벨'' AFTER AD_TIER');

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
CALL _add_column_if_not_exists('WEO_BOARDBBS', 'CONTENTS_MD', 'MEDIUMTEXT NULL COMMENT ''원본 Markdown 텍스트 (편집용)'' AFTER CONTENTS');
CALL _add_column_if_not_exists('WEO_BOARDBBS', 'CONTENT_FORMAT', 'ENUM(''LEGACY'',''MARKDOWN'') DEFAULT ''LEGACY'' COMMENT ''콘텐츠 포맷'' AFTER CONTENTS_MD');


-- =============================================================================
-- 005: Add ad image column to MAIN_AD
-- =============================================================================
CALL _add_column_if_not_exists('MAIN_AD', 'MA_IMG', 'VARCHAR(500) NULL COMMENT ''광고 배너 이미지 URL'' AFTER MA_URL');


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
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_PHONE_PUBLIC', 'ENUM(''Y'',''N'') NOT NULL DEFAULT ''Y''');
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_EMAIL_PUBLIC', 'ENUM(''Y'',''N'') NOT NULL DEFAULT ''Y''');
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
-- 018: Add advertisement publication period
-- =============================================================================
CALL _add_column_if_not_exists('MAIN_AD', 'AD_START_DATE', 'DATETIME NULL COMMENT ''Publication start (UTC)''');
CALL _add_column_if_not_exists('MAIN_AD', 'AD_END_DATE', 'DATETIME NULL COMMENT ''Publication end (UTC)''');


-- =============================================================================
-- 019: Add alumni job position
-- =============================================================================
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_POSITION', 'VARCHAR(100) NULL DEFAULT NULL AFTER USR_BIZ_ADDR');


-- =============================================================================
-- 020: Add donation overwrite flags
-- =============================================================================
CALL _add_column_if_not_exists('DONATION_CONFIG', 'DC_OVERWRITE', 'ENUM(''Y'',''N'') NOT NULL DEFAULT ''N''');
CALL _add_column_if_not_exists('DONATION_SNAPSHOT', 'DS_OVERWRITE', 'ENUM(''Y'',''N'') NOT NULL DEFAULT ''N''');


-- =============================================================================
-- 021: Flip privacy default to private ('N') and backfill existing members
-- =============================================================================
SET @run_021 = (
    SELECT CASE WHEN COUNT(*) = 0 THEN 1 ELSE 0 END
    FROM ALUMNI_SCHEMA_MIGRATION
    WHERE MIGRATION_ID = '021_default_privacy_private'
);
UPDATE WEO_MEMBER SET USR_PHONE_PUBLIC = 'N' WHERE @run_021 = 1 AND USR_PHONE_PUBLIC <> 'N';
UPDATE WEO_MEMBER SET USR_EMAIL_PUBLIC = 'N' WHERE @run_021 = 1 AND USR_EMAIL_PUBLIC <> 'N';
ALTER TABLE WEO_MEMBER
    MODIFY COLUMN USR_PHONE_PUBLIC ENUM('Y','N') NOT NULL DEFAULT 'N';
ALTER TABLE WEO_MEMBER
    MODIFY COLUMN USR_EMAIL_PUBLIC ENUM('Y','N') NOT NULL DEFAULT 'N';
INSERT IGNORE INTO ALUMNI_SCHEMA_MIGRATION (MIGRATION_ID, APPLIED_AT)
SELECT '021_default_privacy_private', NOW()
WHERE @run_021 = 1;


-- =============================================================================
-- 022: Extend subscription billing fields
-- =============================================================================
CALL _add_column_if_not_exists('SUBSCRIPTION', 'BILLING_KEY', 'VARCHAR(64) NULL AFTER PAY_TYPE');
CALL _add_column_if_not_exists('SUBSCRIPTION', 'CARD_NO', 'VARCHAR(32) NULL AFTER BILLING_KEY');
CALL _add_column_if_not_exists('SUBSCRIPTION', 'BILL_DAY', 'TINYINT NULL AFTER NEXT_BILL');
CALL _add_column_if_not_exists('SUBSCRIPTION', 'END_YYYYMM', 'CHAR(6) NULL AFTER BILL_DAY');
CALL _add_column_if_not_exists('SUBSCRIPTION', 'LAST_BILLED_AT', 'DATETIME NULL AFTER END_YYYYMM');
CALL _add_column_if_not_exists('SUBSCRIPTION', 'FAIL_COUNT', 'INT NOT NULL DEFAULT 0 AFTER LAST_BILLED_AT');
CALL _add_column_if_not_exists('SUBSCRIPTION', 'ORDER_SEQ', 'INT NULL AFTER FAIL_COUNT');
CALL _add_index_if_not_exists('SUBSCRIPTION', 'idx_bill_day_status', 'BILL_DAY, STATUS');
CALL _add_index_if_not_exists('SUBSCRIPTION', 'idx_order_seq', 'ORDER_SEQ');


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
-- 024: Normalize alumni department values
-- =============================================================================
UPDATE WEO_MEMBER SET USR_DEPT = '프랑스어' WHERE USR_DEPT IN ('불어과', '불', '프', '프랑스어과');
UPDATE WEO_MEMBER SET USR_DEPT = '독일어' WHERE USR_DEPT IN ('독일어과', '독');
UPDATE WEO_MEMBER SET USR_DEPT = '일본어' WHERE USR_DEPT IN ('일어과', '일', '일본어과');
UPDATE WEO_MEMBER SET USR_DEPT = '중국어' WHERE USR_DEPT IN ('중국어과', '중');
UPDATE WEO_MEMBER SET USR_DEPT = '스페인어' WHERE USR_DEPT IN ('서어과', '스페인어과', '서', '스');
UPDATE WEO_MEMBER SET USR_DEPT = '러시아어' WHERE USR_DEPT IN ('러시아어과', '러');
UPDATE WEO_MEMBER SET USR_DEPT = '영어' WHERE USR_DEPT IN ('영어과', '영');
UPDATE WEO_MEMBER
SET USR_DEPT = NULL
WHERE USR_DEPT IS NOT NULL
  AND USR_DEPT NOT IN ('프랑스어','독일어','일본어','중국어','스페인어','러시아어','영어');


-- =============================================================================
-- 025: Add manual donor-count override
-- =============================================================================
CALL _add_column_if_not_exists('DONATION_CONFIG', 'DC_MANUAL_DONOR_CNT', 'INT NOT NULL DEFAULT 0');


-- =============================================================================
-- 026: Add legacy file registration timestamp
-- =============================================================================
CALL _add_column_if_not_exists('WEO_FILES', 'REG_DATE', 'DATETIME NULL COMMENT ''파일 등록 시각'' AFTER OPEN_YN');


-- =============================================================================
-- 027: Create history entry table and seed existing data
-- =============================================================================
CREATE TABLE IF NOT EXISTS HISTORY_ENTRY (
    HE_SEQ INT AUTO_INCREMENT PRIMARY KEY,
    HE_EVENT_DATE DATE NOT NULL,
    HE_TEXT VARCHAR(500) NOT NULL,
    HE_SORT_ORDER SMALLINT NOT NULL DEFAULT 0,
    REG_DATE DATETIME DEFAULT CURRENT_TIMESTAMP,
    MOD_DATE DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX IDX_DATE (HE_EVENT_DATE, HE_SORT_ORDER)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CALL _add_unique_index_if_not_exists('HISTORY_ENTRY', 'UK_HE_DATE_ORDER', 'HE_EVENT_DATE, HE_SORT_ORDER');

INSERT IGNORE INTO HISTORY_ENTRY (HE_EVENT_DATE, HE_TEXT, HE_SORT_ORDER) VALUES
('2016-12-01', '장학재단 설립 추진을 위한 역대 동문회장단 미팅', 0),
('2016-12-09', '대일외고 장학재단 설립 추진위원회 구성 (위원장: 엄은숙, 부위원장: 이종민)', 1),
('2017-01-12', '대일외고 장학재단 준비위원회 설명회', 0),
('2017-07-14', 'Dream for High School Students (미주 대일외고 장학회) 설립', 1),
('2017-12-04', '대일외고 장학회 창립총회 (회장: 허재혁)', 2),
('2018-12-29', '대일외고 장학회 현판식 및 비전선포식', 0),
('2019-08-26', '비영리민간단체 등록 (서울특별시, 등록번호: 2360)', 0),
('2019-12-31', '기부금대상민간단체 지정 (기획재정부 공고 제2019-219호)', 1),
('2020-03-21', '대일외고 장학회 제2차 현판식', 0),
('2022-07-07', '서울대 최종학 교수 초청 특강', 0),
('2022-08-22', '후원인의 밤', 1),
('2022-09-10', '대일외고 장학회 남춘천CC 골프', 2),
('2022-11-05', '대일외고 장학회 제3차 현판식', 3),
('2023-10-21', '대일외고 장학회 제4차 현판식', 0),
('2024-01-04', '제1회 그랜드마스터클래스 — 우리 아이들에게 펼쳐질 미래 (김승주)', 0),
('2024-07-05', '제2회 그랜드마스터클래스 — 인스타그램, 트렌드를 보는 창 (정다정)', 1),
('2024-08-21', '서울대 최종학 교수 초청 특강', 2),
('2024-10-26', '대일외고 장학회 제5차 현판식', 3),
('2025-01-09', '제3차 조찬 세미나 — 실패 없는 창업 성공법칙 (임상진)', 0),
('2025-02-23', '제1회 재능기부콘서트 — 취업이냐, 창업이냐, 그것이 문제로다 (이준용)', 1),
('2025-05-28', '제2회 재능기부콘서트 — 창업 시 필요한 법률 지식 (허정무)', 2),
('2025-06-26', '후원인의 밤', 3),
('2025-07-28', '제3회 재능기부콘서트 — 절세법과 자산 관리의 노하우 (정용호)', 4),
('2025-09-02', '제4회 그랜드마스터클래스 — 격변의 시대, 자본시장을 읽다 (신창훈)', 5),
('2025-10-23', '제4회 재능기부콘서트 — 안녕하세요? 미래에서 왔습니다! (유효현)', 6),
('2025-10-24', '대일외고 장학회 제6차 현판식', 7);


-- =============================================================================
-- 028: Social auth security and Sign in with Apple
-- =============================================================================
CALL _add_column_if_not_exists('WEO_MEMBER_SOCIAL', 'NMS_EMAIL', 'VARCHAR(255) NULL AFTER NMS_ID');
CALL _add_column_if_not_exists('WEO_MEMBER_SOCIAL', 'NMS_STATUS', 'VARCHAR(20) NOT NULL DEFAULT ''ACTIVE'' AFTER NMS_EMAIL');
CALL _add_column_if_not_exists('WEO_MEMBER_SOCIAL', 'NMS_EMAIL_ENABLED', 'ENUM(''Y'',''N'') NOT NULL DEFAULT ''Y'' AFTER NMS_STATUS');
CALL _add_unique_index_if_not_exists('WEO_MEMBER_SOCIAL', 'UK_USR_PROVIDER', 'USR_SEQ, NMS_GATE');

CREATE TABLE IF NOT EXISTS ALUMNI_MOBILE_REFRESH_TOKEN (
    MRT_JTI VARCHAR(64) NOT NULL PRIMARY KEY,
    USR_SEQ INT NOT NULL,
    MRT_SID VARCHAR(64) NOT NULL,
    EXPIRES_AT DATETIME NOT NULL,
    CREATED_AT DATETIME NOT NULL,
    CONSUMED_AT DATETIME NULL,
    REVOKED_AT DATETIME NULL,
    ROTATED_TO_JTI VARCHAR(64) NULL,
    INDEX IDX_MRT_USR (USR_SEQ),
    INDEX IDX_MRT_SID (MRT_SID),
    INDEX IDX_MRT_EXPIRES (EXPIRES_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_APPLE_NONCE_CHALLENGE (
    CHALLENGE_ID VARCHAR(64) NOT NULL PRIMARY KEY,
    NONCE_HASH CHAR(64) NOT NULL,
    EXPIRES_AT DATETIME NOT NULL,
    CONSUMED_AT DATETIME NULL,
    CREATED_AT DATETIME NOT NULL,
    INDEX IDX_ANC_EXPIRES (EXPIRES_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_APPLE_CODE_REPLAY (
    CODE_HASH CHAR(64) NOT NULL PRIMARY KEY,
    CREATED_AT DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_SOCIAL_CREDENTIAL (
    USR_SEQ INT NOT NULL,
    PROVIDER VARCHAR(10) NOT NULL,
    ENCRYPTED_CREDENTIAL TEXT NOT NULL,
    UPDATED_AT DATETIME NOT NULL,
    PRIMARY KEY (USR_SEQ, PROVIDER)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_SOCIAL_REVOCATION_OUTBOX (
    OUTBOX_ID BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    USR_SEQ INT NOT NULL,
    PROVIDER VARCHAR(10) NOT NULL,
    ACTION VARCHAR(30) NOT NULL,
    STATUS VARCHAR(20) NOT NULL,
    ATTEMPT_COUNT INT NOT NULL DEFAULT 0,
    NEXT_ATTEMPT_AT DATETIME NOT NULL,
    LAST_ERROR VARCHAR(500) NULL,
    CREATED_AT DATETIME NOT NULL,
    UPDATED_AT DATETIME NOT NULL,
    INDEX IDX_SRO_DUE (STATUS, NEXT_ATTEMPT_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- =============================================================================
-- 029: Canonical member phone lookup
-- =============================================================================
UPDATE WEO_MEMBER
SET USR_PHONE = REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '')
WHERE IFNULL(USR_PHONE, '') <> ''
  AND USR_PHONE <> REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '');
CALL _add_index_if_not_exists('WEO_MEMBER', 'IDX_WEO_MEMBER_PHONE', 'USR_PHONE');

-- =============================================================================
-- 030: Separate administrator roles and alumni verification
-- =============================================================================
CREATE TABLE IF NOT EXISTS ALUMNI_ADMIN_ROLE (
    USR_SEQ INT NOT NULL PRIMARY KEY,
    ADMIN_ROLE ENUM('root','operator') NOT NULL,
    CREATED_AT DATETIME NOT NULL,
    UPDATED_AT DATETIME NOT NULL,
    CREATED_BY INT NULL,
    UPDATED_BY INT NULL,
    INDEX IDX_AAR_ROLE (ADMIN_ROLE)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_VERIFICATION (
    USR_SEQ INT NOT NULL PRIMARY KEY,
    STATUS ENUM('unsubmitted','pending','rejected','approved','reapproval_pending') NOT NULL,
    GRADUATION_YEAR SMALLINT UNSIGNED NULL,
    COHORT VARCHAR(20) NULL,
    DEPARTMENT VARCHAR(100) NULL,
    REJECTION_REASON VARCHAR(500) NULL,
    SUBMITTED_AT DATETIME NULL,
    REVIEWED_AT DATETIME NULL,
    REVIEWED_BY INT NULL,
    APPROVED_GRADUATION_YEAR SMALLINT UNSIGNED NULL,
    APPROVED_COHORT VARCHAR(20) NULL,
    APPROVED_DEPARTMENT VARCHAR(100) NULL,
    CREATED_AT DATETIME NOT NULL,
    UPDATED_AT DATETIME NOT NULL,
    INDEX IDX_AV_STATUS_UPDATED (STATUS, UPDATED_AT),
    INDEX IDX_AV_REVIEWER (REVIEWED_BY)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO ALUMNI_ADMIN_ROLE (USR_SEQ, ADMIN_ROLE, CREATED_AT, UPDATED_AT, CREATED_BY, UPDATED_BY)
SELECT USR_SEQ, 'root', NOW(), NOW(), USR_SEQ, USR_SEQ
FROM WEO_MEMBER
WHERE USR_STATUS = 'ZZZ';

INSERT IGNORE INTO ALUMNI_VERIFICATION (
    USR_SEQ, STATUS, GRADUATION_YEAR, COHORT, DEPARTMENT, REJECTION_REASON,
    SUBMITTED_AT, REVIEWED_AT, REVIEWED_BY, APPROVED_GRADUATION_YEAR,
    APPROVED_COHORT, APPROVED_DEPARTMENT, CREATED_AT, UPDATED_AT
)
SELECT
    USR_SEQ,
    CASE
        WHEN USR_STATUS = 'BAA' THEN 'rejected'
        WHEN USR_STATUS = 'BBB' THEN 'pending'
        ELSE 'approved'
    END,
    NULL,
    NULLIF(TRIM(USR_FN), ''),
    NULLIF(TRIM(USR_DEPT), ''),
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    CASE WHEN USR_STATUS IN ('CCC','ZZZ') THEN NULLIF(TRIM(USR_FN), '') ELSE NULL END,
    CASE WHEN USR_STATUS IN ('CCC','ZZZ') THEN NULLIF(TRIM(USR_DEPT), '') ELSE NULL END,
    NOW(),
    NOW()
FROM WEO_MEMBER
WHERE USR_STATUS IN ('BAA','BBB','CCC','ZZZ');

-- =============================================================================
-- 031: One-way member blocking and suppressed-message retention
-- =============================================================================
CREATE TABLE IF NOT EXISTS ALUMNI_MEMBER_BLOCK (
    AMB_SEQ BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    BLOCKER_USR_SEQ INT NOT NULL,
    BLOCKED_USR_SEQ INT NOT NULL,
    CREATED_AT DATETIME NOT NULL,
    UPDATED_AT DATETIME NOT NULL,
    UNIQUE KEY UK_AMB_DIRECTION (BLOCKER_USR_SEQ, BLOCKED_USR_SEQ),
    INDEX IDX_AMB_BLOCKED (BLOCKED_USR_SEQ, BLOCKER_USR_SEQ)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CALL _add_column_if_not_exists('ALUMNI_MESSAGE', 'AM_CLIENT_MESSAGE_ID', 'VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL');
CALL _add_column_if_not_exists('ALUMNI_MESSAGE', 'AM_VISIBLE_RECVR', 'ENUM(''Y'',''N'') NOT NULL DEFAULT ''Y''');
CALL _add_column_if_not_exists('ALUMNI_MESSAGE', 'AM_SUPPRESSION_REASON', 'VARCHAR(30) NULL');
CALL _add_column_if_not_exists('ALUMNI_MESSAGE', 'PURGE_AT', 'DATETIME NULL');
CALL _add_unique_index_if_not_exists('ALUMNI_MESSAGE', 'UK_AM_SENDER_CLIENT', 'AM_SENDER_SEQ, AM_CLIENT_MESSAGE_ID');
CALL _add_index_if_not_exists('ALUMNI_MESSAGE', 'IDX_AM_RECVR_VISIBLE', 'AM_RECVR_SEQ, AM_VISIBLE_RECVR, REG_DATE');
CALL _add_index_if_not_exists('ALUMNI_MESSAGE', 'IDX_AM_PURGE', 'PURGE_AT');

-- =============================================================================
-- 032: Mobile push devices and account-wide preferences
-- =============================================================================
CREATE TABLE IF NOT EXISTS ALUMNI_PUSH_DEVICE (
    APD_SEQ BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    USR_SEQ INT NOT NULL,
    PLATFORM ENUM('android','ios') NOT NULL,
    DEVICE_TOKEN VARCHAR(512) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    APNS_ENVIRONMENT ENUM('sandbox','production') NULL,
    BUNDLE_ID VARCHAR(255) NULL,
    LOCALE VARCHAR(20) NULL,
    LAST_SEEN_AT DATETIME NOT NULL,
    CREATED_AT DATETIME NOT NULL,
    UPDATED_AT DATETIME NOT NULL,
    UNIQUE KEY UK_APD_PLATFORM_TOKEN (PLATFORM, DEVICE_TOKEN),
    INDEX IDX_APD_USER (USR_SEQ, PLATFORM),
    INDEX IDX_APD_LAST_SEEN (LAST_SEEN_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_PUSH_PREFERENCE (
    USR_SEQ INT NOT NULL PRIMARY KEY,
    MESSAGE_ENABLED ENUM('Y','N') NOT NULL DEFAULT 'Y',
    MESSAGE_PREVIEW_ENABLED ENUM('Y','N') NOT NULL DEFAULT 'Y',
    CREATED_AT DATETIME NOT NULL,
    UPDATED_AT DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CALL _add_column_if_not_exists(
    'ALUMNI_PUSH_PREFERENCE',
    'MESSAGE_PREVIEW_ENABLED',
    'ENUM(''Y'',''N'') NOT NULL DEFAULT ''Y'' AFTER MESSAGE_ENABLED'
);

-- =============================================================================
-- 033: Canonical private donation ledger
-- =============================================================================
DROP TEMPORARY TABLE IF EXISTS _MVP_DONATION_PREFLIGHT_ERRORS;
CREATE TEMPORARY TABLE _MVP_DONATION_PREFLIGHT_ERRORS AS
SELECT O_SEQ, 'ambiguous_cancel_or_refund' AS ERROR_CODE
FROM WEO_ORDER
WHERE O_STATUS = 'N'
UNION ALL
SELECT O_SEQ, 'payment_yes_status_not_completed'
FROM WEO_ORDER
WHERE O_PAYMENT = 'Y' AND (O_STATUS IS NULL OR O_STATUS <> 'Y')
UNION ALL
SELECT O_SEQ, 'completed_status_payment_not_yes'
FROM WEO_ORDER
WHERE O_STATUS = 'Y' AND (O_PAYMENT IS NULL OR O_PAYMENT <> 'Y')
UNION ALL
SELECT O_SEQ, 'negative_price'
FROM WEO_ORDER
WHERE O_PRICE < 0
UNION ALL
SELECT O_SEQ, 'negative_paid_amount'
FROM WEO_ORDER
WHERE O_PAY < 0
UNION ALL
SELECT O_SEQ, 'missing_price'
FROM WEO_ORDER
WHERE O_PRICE IS NULL
UNION ALL
SELECT O_SEQ, 'unknown_legacy_status'
FROM WEO_ORDER
WHERE COALESCE(O_STATUS, '') NOT IN ('I','Y','N')
UNION ALL
SELECT O_SEQ, 'unknown_payment_flag'
FROM WEO_ORDER
WHERE COALESCE(O_PAYMENT, '') NOT IN ('Y','N')
UNION ALL
SELECT O_SEQ, 'completed_price_paid_mismatch'
FROM WEO_ORDER
WHERE O_PAYMENT = 'Y' AND O_STATUS = 'Y'
  AND (O_PRICE IS NULL OR O_PAY IS NULL OR O_PRICE <> O_PAY)
UNION ALL
SELECT O_SEQ, 'missing_donation_date'
FROM WEO_ORDER
WHERE COALESCE(O_PAYDATE, O_REGDATE) IS NULL;

SELECT ERROR_CODE, COUNT(*) AS ERROR_COUNT
FROM _MVP_DONATION_PREFLIGHT_ERRORS
GROUP BY ERROR_CODE
ORDER BY ERROR_CODE;

DROP TEMPORARY TABLE IF EXISTS _MVP_DONATION_PREFLIGHT_GUARD;
CREATE TEMPORARY TABLE _MVP_DONATION_PREFLIGHT_GUARD (
    GUARD_ID TINYINT NOT NULL PRIMARY KEY
);
INSERT INTO _MVP_DONATION_PREFLIGHT_GUARD (GUARD_ID) VALUES (1);
INSERT INTO _MVP_DONATION_PREFLIGHT_GUARD (GUARD_ID)
SELECT 1 FROM _MVP_DONATION_PREFLIGHT_ERRORS LIMIT 1;
DROP TEMPORARY TABLE _MVP_DONATION_PREFLIGHT_GUARD;
DROP TEMPORARY TABLE _MVP_DONATION_PREFLIGHT_ERRORS;

CALL _add_column_if_not_exists('WEO_ORDER', 'O_SOURCE', 'VARCHAR(30) NOT NULL DEFAULT ''other''');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_TRANSACTION_NO', 'VARCHAR(191) CHARACTER SET ascii COLLATE ascii_bin NULL');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_COMPOSITE_KEY', 'CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_DONATION_DATE', 'DATE NULL');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_GROSS_AMOUNT', 'BIGINT UNSIGNED NULL');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_REFUNDED_AMOUNT', 'BIGINT UNSIGNED NOT NULL DEFAULT 0');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_NET_RECEIVED_AMOUNT', 'BIGINT UNSIGNED NULL');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_LIFECYCLE_STATUS', 'VARCHAR(30) NOT NULL DEFAULT ''pending''');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_PAYMENT_METHOD', 'VARCHAR(30) NULL');

UPDATE WEO_ORDER
SET O_DONATION_DATE = DATE(COALESCE(O_PAYDATE, O_REGDATE)),
    O_GROSS_AMOUNT = O_PRICE,
    O_REFUNDED_AMOUNT = 0,
    O_NET_RECEIVED_AMOUNT = CASE
        WHEN O_PAYMENT = 'Y' AND O_STATUS = 'Y' THEN O_PAY
        ELSE 0
    END,
    O_LIFECYCLE_STATUS = CASE
        WHEN O_PAYMENT = 'Y' AND O_STATUS = 'Y' THEN 'completed'
        ELSE 'pending'
    END,
    O_PAYMENT_METHOD = CASE UPPER(COALESCE(O_PAY_TYPE, ''))
        WHEN 'CARD' THEN 'card'
        WHEN 'BANK' THEN 'bank'
        WHEN 'VBANK' THEN 'virtual_bank'
        WHEN 'HP' THEN 'mobile'
        WHEN 'ADMS' THEN 'admin'
        ELSE 'other'
    END
WHERE O_GROSS_AMOUNT IS NULL OR O_NET_RECEIVED_AMOUNT IS NULL OR O_DONATION_DATE IS NULL;

CALL _add_unique_index_if_not_exists('WEO_ORDER', 'UK_WO_SOURCE_TRANSACTION', 'O_SOURCE, O_TRANSACTION_NO');
CALL _add_unique_index_if_not_exists('WEO_ORDER', 'UK_WO_COMPOSITE_KEY', 'O_COMPOSITE_KEY');
CALL _add_index_if_not_exists('WEO_ORDER', 'IDX_WO_LIFECYCLE_DATE', 'O_LIFECYCLE_STATUS, O_DONATION_DATE, O_SEQ');

-- =============================================================================
-- 034: Personal-data anonymization and legal retention
-- =============================================================================
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_ANONYMIZED_AT', 'DATETIME NULL');
CALL _add_column_if_not_exists('WEO_MEMBER', 'USR_PURGE_AT', 'DATETIME NULL');
CALL _add_index_if_not_exists('WEO_MEMBER', 'IDX_WM_PERSONAL_PURGE', 'USR_PURGE_AT');

CALL _add_column_if_not_exists('WEO_ORDER', 'O_ACCOUNT_USR_SEQ', 'INT NULL');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_DONOR_NAME', 'VARCHAR(100) NULL');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_DONOR_PHONE', 'VARCHAR(11) CHARACTER SET ascii COLLATE ascii_bin NULL');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_DONOR_COHORT', 'VARCHAR(20) NULL');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_DONOR_DEPARTMENT', 'VARCHAR(100) NULL');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_LEGAL_RETENTION_UNTIL', 'DATETIME NULL');
CALL _add_column_if_not_exists('WEO_ORDER', 'O_ACCOUNT_UNLINKED_AT', 'DATETIME NULL');
CALL _add_index_if_not_exists('WEO_ORDER', 'IDX_WO_LEGAL_RETENTION', 'O_LEGAL_RETENTION_UNTIL, O_ACCOUNT_UNLINKED_AT');
CALL _add_index_if_not_exists('WEO_ORDER', 'IDX_WO_ACCOUNT_DATE', 'O_ACCOUNT_USR_SEQ, O_DONATION_DATE, O_SEQ');

UPDATE WEO_ORDER
SET O_ACCOUNT_USR_SEQ = USR_SEQ
WHERE O_ACCOUNT_USR_SEQ IS NULL
  AND O_ACCOUNT_UNLINKED_AT IS NULL
  AND USR_SEQ > 0;

CALL _add_column_if_not_exists('ALUMNI_MESSAGE', 'AM_SENDER_ACCOUNT_SEQ', 'INT NULL');
CALL _add_column_if_not_exists('ALUMNI_MESSAGE', 'AM_RECVR_ACCOUNT_SEQ', 'INT NULL');
CALL _add_column_if_not_exists('ALUMNI_MESSAGE', 'AM_SENDER_ANONYMIZED_YN', 'ENUM(''Y'',''N'') NOT NULL DEFAULT ''N''');
CALL _add_column_if_not_exists('ALUMNI_MESSAGE', 'AM_RECVR_ANONYMIZED_YN', 'ENUM(''Y'',''N'') NOT NULL DEFAULT ''N''');
CALL _add_index_if_not_exists('ALUMNI_MESSAGE', 'IDX_AM_SENDER_ACCOUNT', 'AM_SENDER_ACCOUNT_SEQ, REG_DATE');
CALL _add_index_if_not_exists('ALUMNI_MESSAGE', 'IDX_AM_RECVR_ACCOUNT', 'AM_RECVR_ACCOUNT_SEQ, REG_DATE');

UPDATE ALUMNI_MESSAGE
SET AM_SENDER_ACCOUNT_SEQ = CASE
        WHEN AM_SENDER_ACCOUNT_SEQ IS NULL AND AM_SENDER_ANONYMIZED_YN = 'N' THEN AM_SENDER_SEQ
        ELSE AM_SENDER_ACCOUNT_SEQ
    END,
    AM_RECVR_ACCOUNT_SEQ = CASE
        WHEN AM_RECVR_ACCOUNT_SEQ IS NULL AND AM_RECVR_ANONYMIZED_YN = 'N' THEN AM_RECVR_SEQ
        ELSE AM_RECVR_ACCOUNT_SEQ
    END
WHERE (AM_SENDER_ACCOUNT_SEQ IS NULL AND AM_SENDER_ANONYMIZED_YN = 'N')
   OR (AM_RECVR_ACCOUNT_SEQ IS NULL AND AM_RECVR_ANONYMIZED_YN = 'N');

-- =============================================================================
-- Cleanup: Drop helper procedures
-- =============================================================================
DROP PROCEDURE IF EXISTS _add_column_if_not_exists;
DROP PROCEDURE IF EXISTS _add_index_if_not_exists;
DROP PROCEDURE IF EXISTS _add_unique_index_if_not_exists;


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

-- 028: Social auth security
SELECT 'WEO_MEMBER_SOCIAL.NMS_EMAIL' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND COLUMN_NAME='NMS_EMAIL';
SELECT 'WEO_MEMBER_SOCIAL.UK_USR_PROVIDER' AS chk, COUNT(DISTINCT INDEX_NAME) AS found FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND INDEX_NAME='UK_USR_PROVIDER';
SELECT 'ALUMNI_MOBILE_REFRESH_TOKEN' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MOBILE_REFRESH_TOKEN';
SELECT 'ALUMNI_APPLE_NONCE_CHALLENGE' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_APPLE_NONCE_CHALLENGE';
SELECT 'ALUMNI_APPLE_CODE_REPLAY' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_APPLE_CODE_REPLAY';
SELECT 'ALUMNI_SOCIAL_CREDENTIAL' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_SOCIAL_CREDENTIAL';
SELECT 'ALUMNI_SOCIAL_REVOCATION_OUTBOX' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_SOCIAL_REVOCATION_OUTBOX';

-- 029: Canonical member phone lookup
SELECT 'WEO_MEMBER.IDX_WEO_MEMBER_PHONE' AS chk, COUNT(DISTINCT INDEX_NAME) AS found FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER' AND INDEX_NAME='IDX_WEO_MEMBER_PHONE';

-- 030: Roles and alumni verification
SELECT 'ALUMNI_ADMIN_ROLE' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_ADMIN_ROLE';
SELECT 'ALUMNI_VERIFICATION' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_VERIFICATION';

-- 031: Blocking and message retention
SELECT 'ALUMNI_MEMBER_BLOCK' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MEMBER_BLOCK';
SELECT 'ALUMNI_MESSAGE.PURGE_AT' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MESSAGE' AND COLUMN_NAME='PURGE_AT';

-- 032: Push
SELECT 'ALUMNI_PUSH_DEVICE' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_PUSH_DEVICE';
SELECT 'ALUMNI_PUSH_PREFERENCE' AS chk, COUNT(*) AS found FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_PUSH_PREFERENCE';

-- 033: Donation ledger
SELECT 'WEO_ORDER.O_NET_RECEIVED_AMOUNT' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_ORDER' AND COLUMN_NAME='O_NET_RECEIVED_AMOUNT';
SELECT 'WEO_ORDER.UK_WO_SOURCE_TRANSACTION' AS chk, COUNT(DISTINCT INDEX_NAME) AS found FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_ORDER' AND INDEX_NAME='UK_WO_SOURCE_TRANSACTION';

-- 034: Retention
SELECT 'WEO_MEMBER.USR_PURGE_AT' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER' AND COLUMN_NAME='USR_PURGE_AT';
SELECT 'WEO_ORDER.O_LEGAL_RETENTION_UNTIL' AS chk, COUNT(*) AS found FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_ORDER' AND COLUMN_NAME='O_LEGAL_RETENTION_UNTIL';

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
