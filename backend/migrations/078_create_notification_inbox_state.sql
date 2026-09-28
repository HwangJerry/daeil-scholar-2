-- Migration 078: Per-member "last seen" marker for the in-app notification inbox.
-- The inbox itself is derived from published notices (WEO_BOARDBBS GATE='NOTICE'),
-- so the only per-member state is the newest notice SEQ the member's app has shown.
-- A notice with SEQ > LAST_SEEN_POST_SEQ is unread; a member without a row has
-- seen nothing (0). A SEQ marker, unlike a timestamp, cannot mark as seen a notice
-- published between the app loading the list and reporting it seen.
-- The inbox query reuses IDX_BBS_FEED (GATE, OPEN_YN, SEQ); no new index is needed.
-- Target: MariaDB 10.1.38.

CREATE TABLE IF NOT EXISTS ALUMNI_NOTIFICATION_INBOX_STATE (
    USR_SEQ            INT NOT NULL,
    LAST_SEEN_POST_SEQ INT NOT NULL DEFAULT 0,
    UPD_DATE           DATETIME NOT NULL,
    PRIMARY KEY (USR_SEQ)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Rollback:
-- DROP TABLE ALUMNI_NOTIFICATION_INBOX_STATE;
