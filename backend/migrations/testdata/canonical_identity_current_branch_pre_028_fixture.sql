-- Minimal current-branch schema immediately before 028_social_auth_security.sql.
-- Synthetic empty data only; used to expose the incompatible 028 -> 036 lineage.
DROP DATABASE IF EXISTS canonical_identity_test;
CREATE DATABASE canonical_identity_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE canonical_identity_test;

CREATE TABLE _migration_history (
    filename VARCHAR(255) CHARACTER SET ascii NOT NULL PRIMARY KEY,
    sha256 CHAR(64) CHARACTER SET ascii NOT NULL,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE WEO_MEMBER_SOCIAL (
    NMS_SEQ INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    USR_SEQ INT NOT NULL,
    NMS_GATE VARCHAR(20) NOT NULL,
    NMS_ID VARCHAR(191) NOT NULL,
    NMS_DATE DATETIME NOT NULL,
    INDEX IDX_SOCIAL_USR (USR_SEQ)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;
