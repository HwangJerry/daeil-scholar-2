-- Existing decisions need a fresh review; never infer a version for historical evidence.
ALTER TABLE ALUMNI_DONATION_RETENTION ADD COLUMN SOURCE_FINGERPRINT CHAR(64) NOT NULL DEFAULT '';
