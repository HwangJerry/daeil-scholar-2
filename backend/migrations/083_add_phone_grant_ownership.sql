-- Additive ownership: do not infer owners for legacy SMS rows or use a foreign key
-- that loses the owner when the member is erased. Apply before the new server.
ALTER TABLE ALUMNI_PHONE_VERIFICATION
    ADD COLUMN CONSUMED_USR_SEQ INT NULL,
    ADD COLUMN CONSUMED_AT DATETIME NULL,
    ADD INDEX idx_phone_grant_owner (CONSUMED_USR_SEQ);
ALTER TABLE ALUMNI_ERASURE_TARGET
    ADD COLUMN WAIT_COUNT BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN WAIT_UNTIL DATETIME NULL;
