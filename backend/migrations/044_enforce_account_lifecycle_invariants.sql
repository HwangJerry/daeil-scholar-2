-- Migration 044: Database authorities for signup phone ownership and account lifecycle.
-- Application planned: foundation for account unification (email/Kakao/Apple linking).
-- Target: MariaDB 10.1.38. Apply after canonical identity cutover migration 043.

-- Account deletion relies on rollback across every touched table. Fail before
-- any DDL if the deployed schema is missing one or still uses a non-transactional engine.
DROP TEMPORARY TABLE IF EXISTS _044_engine_errors;
CREATE TEMPORARY TABLE _044_engine_errors AS
SELECT required.TABLE_NAME
FROM (
    SELECT 'WEO_MEMBER' TABLE_NAME UNION ALL SELECT 'WEO_ORDER' UNION ALL
    SELECT 'ALUMNI_MESSAGE' UNION ALL SELECT 'WEO_MEMBER_LOG' UNION ALL
    SELECT 'WEO_MEMBER_SOCIAL' UNION ALL SELECT 'ALUMNI_MOBILE_REFRESH_TOKEN' UNION ALL
    SELECT 'ALUMNI_MOBILE_DEVICE_TOKEN' UNION ALL SELECT 'ALUMNI_PUSH_PREFERENCE' UNION ALL
    SELECT 'ALUMNI_MEMBER_BLOCK' UNION ALL SELECT 'ALUMNI_USER_TAG' UNION ALL
    SELECT 'ALUMNI_ADMIN_ROLE' UNION ALL SELECT 'ALUMNI_VERIFICATION' UNION ALL
    SELECT 'ALUMNI_SOCIAL_CREDENTIAL' UNION ALL SELECT 'ALUMNI_SOCIAL_REVOCATION_OUTBOX' UNION ALL
    SELECT 'AUTH_ACCOUNT_STATE' UNION ALL SELECT 'AUTH_IDENTITY' UNION ALL
    SELECT 'AUTH_PASSWORD_CREDENTIAL' UNION ALL SELECT 'AUTH_SESSION_FAMILY'
) required
LEFT JOIN information_schema.TABLES deployed
  ON deployed.TABLE_SCHEMA = DATABASE()
 AND deployed.TABLE_NAME = required.TABLE_NAME
 AND deployed.ENGINE = 'InnoDB'
WHERE deployed.TABLE_NAME IS NULL;

DROP TEMPORARY TABLE IF EXISTS _044_engine_guard;
CREATE TEMPORARY TABLE _044_engine_guard (GUARD_ID TINYINT NOT NULL PRIMARY KEY);
INSERT INTO _044_engine_guard (GUARD_ID) VALUES (1);
INSERT INTO _044_engine_guard (GUARD_ID)
SELECT 1 FROM _044_engine_errors LIMIT 1;
DROP TEMPORARY TABLE _044_engine_guard;
DROP TEMPORARY TABLE _044_engine_errors;

-- Fail before any persistent DDL when an older application version has left
-- multiple open attempts. Operators can retain the newest row and mark older
-- rows DELIVERED/FAILED, then safely rerun this migration.
DROP TEMPORARY TABLE IF EXISTS _044_open_outbox_conflicts;
CREATE TEMPORARY TABLE _044_open_outbox_conflicts AS
SELECT USR_SEQ, PROVIDER, ACTION
FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX
WHERE STATUS IN ('PENDING','PROCESSING')
GROUP BY USR_SEQ, PROVIDER, ACTION
HAVING COUNT(*) > 1;

DROP TEMPORARY TABLE IF EXISTS _044_open_outbox_guard;
CREATE TEMPORARY TABLE _044_open_outbox_guard (GUARD_ID TINYINT NOT NULL PRIMARY KEY);
INSERT INTO _044_open_outbox_guard (GUARD_ID) VALUES (1);
INSERT INTO _044_open_outbox_guard (GUARD_ID)
SELECT 1 FROM _044_open_outbox_conflicts LIMIT 1;
DROP TEMPORARY TABLE _044_open_outbox_guard;
DROP TEMPORARY TABLE _044_open_outbox_conflicts;

-- MariaDB 10.1 has no REGEXP_REPLACE. Build the same ASCII-digits-only value
-- as model.NormalizePhoneNumber instead of maintaining a punctuation allowlist.
DROP TEMPORARY TABLE IF EXISTS _044_phone_claim_source;
CREATE TEMPORARY TABLE _044_phone_claim_source (
    ACCOUNT_ID INT NOT NULL PRIMARY KEY,
    CANONICAL_PHONE VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL
);

DELIMITER //
CREATE PROCEDURE _044_build_phone_claim_source()
BEGIN
    DECLARE done INT DEFAULT 0;
    DECLARE account_id INT;
    DECLARE raw_phone VARCHAR(255);
    DECLARE canonical_phone VARCHAR(32);
    DECLARE character_index INT;
    DECLARE candidate_character CHAR(1);
    DECLARE member_cursor CURSOR FOR
        SELECT USR_SEQ, COALESCE(USR_PHONE, '')
        FROM WEO_MEMBER
        WHERE COALESCE(USR_STATUS, '') <> 'AAA'
        ORDER BY USR_SEQ;
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = 1;

    OPEN member_cursor;
    member_loop: LOOP
        FETCH member_cursor INTO account_id, raw_phone;
        IF done = 1 THEN
            LEAVE member_loop;
        END IF;
        SET canonical_phone = '';
        SET character_index = 1;
        WHILE character_index <= CHAR_LENGTH(raw_phone) DO
            SET candidate_character = SUBSTRING(raw_phone, character_index, 1);
            IF ASCII(candidate_character) BETWEEN ASCII('0') AND ASCII('9') THEN
                SET canonical_phone = CONCAT(canonical_phone, candidate_character);
            END IF;
            SET character_index = character_index + 1;
        END WHILE;
        INSERT INTO _044_phone_claim_source (ACCOUNT_ID, CANONICAL_PHONE)
        VALUES (account_id, canonical_phone);
    END LOOP;
    CLOSE member_cursor;
END //
DELIMITER ;

CALL _044_build_phone_claim_source();
DROP PROCEDURE _044_build_phone_claim_source;

DROP TEMPORARY TABLE IF EXISTS _044_phone_claim_conflicts;
CREATE TEMPORARY TABLE _044_phone_claim_conflicts AS
SELECT CANONICAL_PHONE
FROM _044_phone_claim_source
GROUP BY CANONICAL_PHONE
HAVING CHAR_LENGTH(CANONICAL_PHONE) NOT BETWEEN 7 AND 15 OR COUNT(*) > 1;

DROP TEMPORARY TABLE IF EXISTS _044_phone_claim_guard;
CREATE TEMPORARY TABLE _044_phone_claim_guard (GUARD_ID TINYINT NOT NULL PRIMARY KEY);
INSERT INTO _044_phone_claim_guard (GUARD_ID) VALUES (1);
-- Deliberately raises duplicate-key error 1062 for empty or duplicate claims.
INSERT INTO _044_phone_claim_guard (GUARD_ID)
SELECT 1 FROM _044_phone_claim_conflicts LIMIT 1;
DROP TEMPORARY TABLE _044_phone_claim_guard;
DROP TEMPORARY TABLE _044_phone_claim_conflicts;

CREATE TABLE AUTH_PHONE_CLAIM (
    CANONICAL_PHONE VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    ACCOUNT_ID      INT NOT NULL,
    CREATED_AT      DATETIME NOT NULL,
    PRIMARY KEY (CANONICAL_PHONE),
    UNIQUE KEY UQ_AUTH_PHONE_CLAIM_ACCOUNT (ACCOUNT_ID),
    CONSTRAINT FK_AUTH_PHONE_CLAIM_ACCOUNT
        FOREIGN KEY (ACCOUNT_ID) REFERENCES WEO_MEMBER (USR_SEQ)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO AUTH_PHONE_CLAIM (CANONICAL_PHONE, ACCOUNT_ID, CREATED_AT)
SELECT CANONICAL_PHONE, ACCOUNT_ID, NOW()
FROM _044_phone_claim_source;

DROP TEMPORARY TABLE _044_phone_claim_source;

-- Legal-retention snapshots must hold the same canonical value accepted by
-- AUTH_PHONE_CLAIM, including international numbers up to 15 digits.
ALTER TABLE WEO_ORDER MODIFY O_DONOR_PHONE VARCHAR(32) NULL;

-- Multiple completed attempts are retained for audit, while only one open
-- revocation may exist for an account/provider/action tuple.
ALTER TABLE ALUMNI_SOCIAL_REVOCATION_OUTBOX
    ADD COLUMN OPEN_ACTION VARCHAR(30)
        GENERATED ALWAYS AS (
            CASE WHEN STATUS IN ('PENDING','PROCESSING') THEN ACTION ELSE NULL END
        ) PERSISTENT,
    ADD UNIQUE KEY UQ_SRO_OPEN_ACTION (USR_SEQ, PROVIDER, OPEN_ACTION);
