-- Additive only. Existing requests have no schedule and require operator review.
-- No member data is changed and no queued deletion is activated by this migration.
CREATE TABLE IF NOT EXISTS ALUMNI_ERASURE_SCHEDULE (
 REQUEST_ID BIGINT NOT NULL PRIMARY KEY,
 SCHEDULED_AT DATETIME NOT NULL,
 EXPEDITED_AT DATETIME NULL,
 EXPEDITED_BY INT NULL,
 CREATED_AT DATETIME NOT NULL,
 KEY IDX_ERASURE_SCHEDULE (SCHEDULED_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
