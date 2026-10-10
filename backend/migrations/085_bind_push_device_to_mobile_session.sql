-- Existing rows stay NULL: session ownership cannot be backfilled safely.
-- Apply before the server; NULL legacy rows are suppressed until mobile re-registration.
SET @push_sid_ddl = IF(
    EXISTS (SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MOBILE_DEVICE_TOKEN' AND COLUMN_NAME='SESSION_SID'),
    'SELECT 1',
    'ALTER TABLE ALUMNI_MOBILE_DEVICE_TOKEN ADD COLUMN SESSION_SID CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NULL AFTER USR_SEQ');
PREPARE push_sid_statement FROM @push_sid_ddl;
EXECUTE push_sid_statement;
DEALLOCATE PREPARE push_sid_statement;
SET @push_sid_ddl = IF(
    EXISTS (SELECT 1 FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MOBILE_DEVICE_TOKEN' AND INDEX_NAME='IDX_MDT_ACCOUNT_SESSION'),
    'SELECT 1',
    'ALTER TABLE ALUMNI_MOBILE_DEVICE_TOKEN ADD INDEX IDX_MDT_ACCOUNT_SESSION (USR_SEQ,SESSION_SID)');
PREPARE push_sid_statement FROM @push_sid_ddl;
EXECUTE push_sid_statement;
DEALLOCATE PREPARE push_sid_statement;
SET @push_sid_ddl = NULL;
