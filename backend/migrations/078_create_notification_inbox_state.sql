-- Migration 078: Per-member "last seen" marker for the in-app notification inbox.
-- The inbox itself is derived from published notices (WEO_BOARDBBS GATE='NOTICE'),
-- so the only per-member state is when the member last opened it. A notice
-- registered after LAST_SEEN_AT is unread; a member without a row has seen nothing.
-- Both timestamps are written by the application with NOW(), the same clock that
-- stamps WEO_BOARDBBS.REG_DATE, so the comparison never mixes time zones.
-- The inbox query reuses IDX_BBS_FEED (GATE, OPEN_YN, SEQ); no new index is needed.
-- Target: MariaDB 10.1.38.

CREATE TABLE IF NOT EXISTS ALUMNI_NOTIFICATION_INBOX_STATE (
    USR_SEQ      INT NOT NULL,
    LAST_SEEN_AT DATETIME NOT NULL,
    UPD_DATE     DATETIME NOT NULL,
    PRIMARY KEY (USR_SEQ)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Rollback:
-- DROP TABLE ALUMNI_NOTIFICATION_INBOX_STATE;
