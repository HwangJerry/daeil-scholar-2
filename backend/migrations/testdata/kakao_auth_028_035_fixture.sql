-- Production-shape Kakao auth migration fixture.
-- Production metadata captured read-only on 2026-08-04; no row identifiers or values copied.
-- Prerequisite migration source SHA-256:
-- 028 44322db83275b2f5445a79a5b2b9d0d2671e3674d5f7c704440377dde9146863
-- 029 77e3283ff6476846f1407590742f365ec45c10ad492a8008453a9a8b5fe62461
-- 030 13db8bbdf57e05fa5b8cd4faa760ee8420ec0d2d3f416b26a6dabd11d3d346a9
-- 031 26d69c94f5a55531d49ad41bbfe0e4e953bc18c871227d9ee402044d5734340c
-- 032 5247277caaf8afbd0cc2053270898eef335e6e75a18b377228453e95887e1969
-- 033 355a5c503bbc094a096c564e50393d76fff6e352508aab205bca9ba13322cdbf
-- 034 6732ded58bbf19f03fbccb90c82198304be9d7819d89d3e6a4decdd5ea7c1522
-- 035 1bbd046048a322f39cfbc313269528900bd59df1df9320cacb3e6fd10dce4500

CREATE TABLE WEO_MEMBER (
    USR_SEQ    INT NOT NULL PRIMARY KEY,
    USR_ID     VARCHAR(50) NOT NULL,
    USR_NAME   VARCHAR(100) NOT NULL,
    USR_STATUS CHAR(3) NOT NULL,
    USR_PHONE  VARCHAR(20) NULL,
    USR_FN     VARCHAR(20) NULL,
    USR_DEPT   VARCHAR(100) NULL,
    USR_EMAIL  VARCHAR(255) NULL,
    USR_NICK   VARCHAR(100) NULL,
    USR_PHOTO  VARCHAR(500) NULL
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

CREATE TABLE WEO_MEMBER_SOCIAL (
    SEQ           INT(10) NOT NULL AUTO_INCREMENT,
    NMS_GATE      CHAR(2) NOT NULL,
    USR_SEQ       INT(10) NOT NULL,
    NMS_ID        VARCHAR(50) NOT NULL,
    NMS_EMAIL     VARCHAR(100) NOT NULL,
    NMS_TOKEN     VARCHAR(100) NULL,
    NMS_THUMNAIL  VARCHAR(100) NULL,
    NMS_STATUS    ENUM('Y','N') NULL DEFAULT 'Y',
    REG_DATE      DATETIME NULL,
    OUT_DATE      DATETIME NULL,
    PRIMARY KEY (SEQ)
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

INSERT INTO WEO_MEMBER (
    USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_FN, USR_DEPT, USR_EMAIL
) VALUES
    (101, 'fixture-baa', 'Fixture BAA', 'BAA', '10', 'English', 'baa@example.invalid'),
    (102, 'fixture-bbb', 'Fixture BBB', 'BBB', '11', 'Chinese', 'bbb@example.invalid'),
    (103, 'fixture-ccc', 'Fixture CCC', 'CCC', '12', 'Japanese', 'ccc@example.invalid'),
    (104, 'fixture-zzz', 'Fixture ZZZ', 'ZZZ', '13', 'German', 'zzz@example.invalid'),
    (105, 'fixture-aaa', 'Fixture AAA', 'AAA', '14', 'French', 'aaa@example.invalid');

INSERT INTO WEO_MEMBER_SOCIAL (
    NMS_GATE, USR_SEQ, NMS_ID, NMS_EMAIL, NMS_STATUS, REG_DATE
) VALUES
    ('KT', 103, 'fixture-subject-1', 'provider1@example.invalid', 'Y', NOW()),
    ('KT', 104, 'fixture-subject-2', 'provider2@example.invalid', 'Y', NOW());
