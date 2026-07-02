-- 033_backfill_android_push_token_metadata_and_length.sql
-- Android FCM tokens can be longer than APNs tokens and do not carry APNs routing metadata.
-- DEVICE_TOKEN uses ASCII binary collation so the full UNIQUE key remains 512 bytes on older InnoDB row formats.

ALTER TABLE ALUMNI_MOBILE_DEVICE_TOKEN
    MODIFY DEVICE_TOKEN VARCHAR(512) CHARACTER SET ascii COLLATE ascii_bin NOT NULL;

UPDATE ALUMNI_MOBILE_DEVICE_TOKEN
SET APNS_ENVIRONMENT = NULL,
    BUNDLE_ID = NULL,
    UPDATED_AT = NOW()
WHERE PLATFORM = 'android'
  AND (APNS_ENVIRONMENT IS NOT NULL OR BUNDLE_ID IS NOT NULL);
