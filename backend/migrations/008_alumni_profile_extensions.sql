-- Migration 008: Alumni profile extensions — job categories, user tags, business info
-- Target: MariaDB 10.1.38

-- 1) 업종 카테고리 마스터 테이블
CREATE TABLE ALUMNI_JOB_CATEGORY (
    AJC_SEQ   INT AUTO_INCREMENT PRIMARY KEY,
    AJC_NAME  VARCHAR(50) NOT NULL,
    AJC_COLOR VARCHAR(7) DEFAULT '#4F46E5',
    AJC_INDX  INT DEFAULT 0,
    OPEN_YN   ENUM('Y','N') DEFAULT 'Y',
    REG_DATE  DATETIME NULL,
    UNIQUE KEY UK_NAME (AJC_NAME)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 2) 사용자 태그 테이블
CREATE TABLE ALUMNI_USER_TAG (
    AUT_SEQ  INT AUTO_INCREMENT PRIMARY KEY,
    USR_SEQ  INT NOT NULL,
    AUT_TAG  VARCHAR(30) NOT NULL,
    AUT_INDX INT DEFAULT 0,
    REG_DATE DATETIME NULL,
    INDEX IDX_USR_SEQ (USR_SEQ),
    UNIQUE KEY UK_USR_TAG (USR_SEQ, AUT_TAG)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 3) WEO_MEMBER 사업장 정보 컬럼 추가
ALTER TABLE WEO_MEMBER
    ADD COLUMN USR_BIZ_NAME VARCHAR(100) NULL AFTER USR_PHOTO,
    ADD COLUMN USR_BIZ_DESC VARCHAR(200) NULL AFTER USR_BIZ_NAME,
    ADD COLUMN USR_BIZ_ADDR VARCHAR(200) NULL AFTER USR_BIZ_DESC,
    ADD COLUMN USR_JOB_CAT  INT NULL AFTER USR_BIZ_ADDR;

-- 4) 업종 카테고리 시드 데이터
INSERT INTO ALUMNI_JOB_CATEGORY (AJC_NAME, AJC_COLOR, AJC_INDX, OPEN_YN, REG_DATE) VALUES
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
