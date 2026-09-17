-- Phone ownership verification for signup: one-time SMS codes and the short-lived
-- grant token that proves a phone number was verified before registration.
-- Codes and grant tokens are stored as SHA-256 hashes only; plaintext never lands here.
-- Compatible with MariaDB 10.1. Apply before deploying the phone verification endpoints.
CREATE TABLE IF NOT EXISTS ALUMNI_PHONE_VERIFICATION (
    APV_SEQ BIGINT NOT NULL AUTO_INCREMENT,
    APV_ID CHAR(32) NOT NULL,
    PHONE VARCHAR(20) NOT NULL,
    CODE_HASH CHAR(64) NOT NULL,
    GRANT_TOKEN_HASH CHAR(64) NULL,
    ATTEMPTS INT NOT NULL DEFAULT 0,
    VERIFIED_YN CHAR(1) NOT NULL DEFAULT 'N',
    CONSUMED_YN CHAR(1) NOT NULL DEFAULT 'N',
    EXPIRES_AT DATETIME NOT NULL,
    GRANT_EXPIRES_AT DATETIME NULL,
    REG_DATE DATETIME NOT NULL,
    PRIMARY KEY (APV_SEQ),
    UNIQUE KEY uq_phone_verification_id (APV_ID),
    UNIQUE KEY uq_phone_verification_grant (GRANT_TOKEN_HASH),
    KEY idx_phone_verification_throttle (PHONE, REG_DATE),
    KEY idx_phone_verification_retention (REG_DATE)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
