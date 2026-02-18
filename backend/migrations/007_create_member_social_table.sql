-- Migration 007: Create WEO_MEMBER_SOCIAL table for OAuth account linking
-- Target: MariaDB 10.1.38
--
-- Columns derived from:
--   model/user.go MemberSocial struct
--   repository/auth_repo.go FindMemberByKakaoID / InsertSocialLink

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
