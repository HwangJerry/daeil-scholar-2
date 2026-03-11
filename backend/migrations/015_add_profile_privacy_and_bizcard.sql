-- 015_add_profile_privacy_and_bizcard.sql
-- Adds phone/email public visibility toggles and biz card image column to WEO_MEMBER

ALTER TABLE WEO_MEMBER
  ADD COLUMN USR_PHONE_PUBLIC ENUM('Y','N') NOT NULL DEFAULT 'Y',
  ADD COLUMN USR_EMAIL_PUBLIC ENUM('Y','N') NOT NULL DEFAULT 'Y',
  ADD COLUMN USR_BIZ_CARD VARCHAR(500) NULL;
