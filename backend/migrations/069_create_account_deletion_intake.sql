-- Operator intake of deletion requests received without the app (for example by email).
-- MariaDB 10.1.38. One row per request registered on a member's behalf: who
-- registered it and the identity-check reference, never personal data.
-- Rows are removed together with the completed request by the retention job.
CREATE TABLE IF NOT EXISTS ALUMNI_ACCOUNT_DELETION_INTAKE (
    REQUEST_ID         BIGINT NOT NULL PRIMARY KEY,
    OPERATOR_SEQ       INT NOT NULL,
    EVIDENCE_REFERENCE VARCHAR(200) NOT NULL,
    CREATED_AT         DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Rollback (only after reverting the application binary):
-- DROP TABLE ALUMNI_ACCOUNT_DELETION_INTAKE;
