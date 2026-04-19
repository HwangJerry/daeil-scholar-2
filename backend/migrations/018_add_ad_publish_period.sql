-- 018_add_ad_publish_period.sql — Add publication period columns to MAIN_AD
ALTER TABLE MAIN_AD
  ADD COLUMN AD_START_DATE DATETIME NULL COMMENT 'Publication start (UTC)',
  ADD COLUMN AD_END_DATE   DATETIME NULL COMMENT 'Publication end (UTC)';
