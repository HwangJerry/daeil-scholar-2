-- Migration 030: Separate administrator roles and alumni verification state.
-- Target: MariaDB 10.1.38. Apply manually after a backup.
-- Existing ZZZ administrators are backfilled as root to preserve current access.
-- Existing BAA/BBB/CCC/ZZZ members retain their legacy approval meaning; source
-- records do not contain reliable submission/review timestamps, so those stay NULL.
-- BAA preserves the rejected state with a NULL reason because legacy data has no
-- reliable rejection-reason source; administrators may add a reason on re-review.

CREATE TABLE IF NOT EXISTS ALUMNI_ADMIN_ROLE (
    USR_SEQ       INT NOT NULL PRIMARY KEY,
    ADMIN_ROLE    ENUM('root','operator') NOT NULL,
    CREATED_AT    DATETIME NOT NULL,
    UPDATED_AT    DATETIME NOT NULL,
    CREATED_BY    INT NULL,
    UPDATED_BY    INT NULL,
    INDEX IDX_AAR_ROLE (ADMIN_ROLE)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ALUMNI_VERIFICATION (
    USR_SEQ                     INT NOT NULL PRIMARY KEY,
    STATUS                      ENUM('unsubmitted','pending','rejected','approved','reapproval_pending') NOT NULL,
    GRADUATION_YEAR             SMALLINT UNSIGNED NULL,
    COHORT                      VARCHAR(20) NULL,
    DEPARTMENT                  VARCHAR(100) NULL,
    REJECTION_REASON            VARCHAR(500) NULL,
    SUBMITTED_AT                DATETIME NULL,
    REVIEWED_AT                 DATETIME NULL,
    REVIEWED_BY                 INT NULL,
    APPROVED_GRADUATION_YEAR    SMALLINT UNSIGNED NULL,
    APPROVED_COHORT             VARCHAR(20) NULL,
    APPROVED_DEPARTMENT         VARCHAR(100) NULL,
    CREATED_AT                  DATETIME NOT NULL,
    UPDATED_AT                  DATETIME NOT NULL,
    INDEX IDX_AV_STATUS_UPDATED (STATUS, UPDATED_AT),
    INDEX IDX_AV_REVIEWER (REVIEWED_BY)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO ALUMNI_ADMIN_ROLE (
    USR_SEQ, ADMIN_ROLE, CREATED_AT, UPDATED_AT, CREATED_BY, UPDATED_BY
)
SELECT USR_SEQ, 'root', NOW(), NOW(), USR_SEQ, USR_SEQ
FROM WEO_MEMBER
WHERE USR_STATUS = 'ZZZ';

INSERT IGNORE INTO ALUMNI_VERIFICATION (
    USR_SEQ,
    STATUS,
    GRADUATION_YEAR,
    COHORT,
    DEPARTMENT,
    REJECTION_REASON,
    SUBMITTED_AT,
    REVIEWED_AT,
    REVIEWED_BY,
    APPROVED_GRADUATION_YEAR,
    APPROVED_COHORT,
    APPROVED_DEPARTMENT,
    CREATED_AT,
    UPDATED_AT
)
SELECT
    USR_SEQ,
    CASE
        WHEN USR_STATUS = 'BAA' THEN 'rejected'
        WHEN USR_STATUS = 'BBB' THEN 'pending'
        ELSE 'approved'
    END,
    NULL,
    NULLIF(TRIM(USR_FN), ''),
    NULLIF(TRIM(USR_DEPT), ''),
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    CASE WHEN USR_STATUS IN ('CCC','ZZZ') THEN NULLIF(TRIM(USR_FN), '') ELSE NULL END,
    CASE WHEN USR_STATUS IN ('CCC','ZZZ') THEN NULLIF(TRIM(USR_DEPT), '') ELSE NULL END,
    NOW(),
    NOW()
FROM WEO_MEMBER
WHERE USR_STATUS IN ('BAA','BBB','CCC','ZZZ');
