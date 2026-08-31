-- canonical_identity_schema_contract_v1
-- Executed after candidate migrations 040-042 on MariaDB 10.1.38.
SET @canonical_contract_failures = 0;
SET SESSION group_concat_max_len = 1048576;
SET @canonical_tables = 'AUTH_ACCOUNT_STATE,AUTH_CONSENT,AUTH_EMAIL_VERIFICATION,AUTH_IDENTITY,AUTH_IDENTITY_MIGRATION_JOURNAL,AUTH_IDENTITY_MIGRATION_RUN,AUTH_PASSWORD_CREDENTIAL,AUTH_PROVIDER_CREDENTIAL,AUTH_PROVIDER_REVOKE_OUTBOX,AUTH_SESSION_FAMILY,AUTH_SIGNUP_CONTINUATION';

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.TABLES
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME IN (
          'AUTH_ACCOUNT_STATE',
          'AUTH_IDENTITY',
          'AUTH_PASSWORD_CREDENTIAL',
          'AUTH_PROVIDER_CREDENTIAL',
          'AUTH_EMAIL_VERIFICATION',
          'AUTH_SIGNUP_CONTINUATION',
          'AUTH_CONSENT',
          'AUTH_SESSION_FAMILY',
          'AUTH_PROVIDER_REVOKE_OUTBOX',
          'AUTH_IDENTITY_MIGRATION_RUN',
          'AUTH_IDENTITY_MIGRATION_JOURNAL'
      )
      AND ENGINE = 'InnoDB'
) = 11, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'AUTH_IDENTITY'
      AND COLUMN_NAME = 'PROVIDER'
      AND COLUMN_TYPE = "enum('EMAIL','KAKAO','APPLE','LOCAL_USERNAME')"
      AND IS_NULLABLE = 'NO'
) = 1, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'AUTH_IDENTITY'
      AND COLUMN_NAME = 'NORMALIZED_EMAIL'
      AND IS_NULLABLE = 'YES'
) = 1, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM (
        SELECT INDEX_NAME
        FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'AUTH_IDENTITY'
        GROUP BY INDEX_NAME, NON_UNIQUE
        HAVING NON_UNIQUE = 0
           AND GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ',') = 'PROVIDER,SUBJECT_KEY'
    ) AS provider_subject_indexes
) = 1, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM (
        SELECT INDEX_NAME
        FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'AUTH_IDENTITY'
        GROUP BY INDEX_NAME, NON_UNIQUE
        HAVING NON_UNIQUE = 0
           AND GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ',') = 'ACCOUNT_ID,PROVIDER'
    ) AS forbidden_account_provider_indexes
) = 0, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM (
        SELECT INDEX_NAME
        FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE()
          AND TABLE_NAME = 'AUTH_IDENTITY'
        GROUP BY INDEX_NAME, NON_UNIQUE
        HAVING NON_UNIQUE = 0
           AND GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ',') = 'NORMALIZED_EMAIL'
    ) AS normalized_email_indexes
) = 1, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.KEY_COLUMN_USAGE
    WHERE CONSTRAINT_SCHEMA = DATABASE()
      AND TABLE_NAME IN ('AUTH_ACCOUNT_STATE','AUTH_IDENTITY','AUTH_CONSENT')
      AND COLUMN_NAME = 'ACCOUNT_ID'
      AND REFERENCED_TABLE_NAME = 'WEO_MEMBER'
      AND REFERENCED_COLUMN_NAME = 'USR_SEQ'
) = 3, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.TABLE_CONSTRAINTS
    WHERE CONSTRAINT_SCHEMA = DATABASE()
      AND TABLE_NAME = 'AUTH_IDENTITY'
      AND CONSTRAINT_TYPE = 'FOREIGN KEY'
) = 1, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.KEY_COLUMN_USAGE
    WHERE CONSTRAINT_SCHEMA = DATABASE()
      AND TABLE_NAME = 'AUTH_SESSION_FAMILY'
      AND CONSTRAINT_NAME = 'FK_AUTH_SESSION_FAMILY_IDENTITY_ACCOUNT'
      AND REFERENCED_TABLE_NAME = 'AUTH_IDENTITY'
      AND COLUMN_NAME IN ('IDENTITY_ID','ACCOUNT_ID')
) = 2, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME IN ('AUTH_EMAIL_VERIFICATION','AUTH_SIGNUP_CONTINUATION')
      AND COLUMN_NAME = 'TOKEN_HASH'
      AND DATA_TYPE = 'char'
      AND CHARACTER_MAXIMUM_LENGTH = 64
      AND IS_NULLABLE = 'NO'
) = 2, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME IN ('AUTH_EMAIL_VERIFICATION','AUTH_SIGNUP_CONTINUATION')
      AND (COLUMN_NAME = 'TOKEN' OR COLUMN_NAME LIKE 'RAW_TOKEN%')
) = 0, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'AUTH_PROVIDER_CREDENTIAL'
      AND COLUMN_NAME IN ('KEY_ID','NONCE_BYTES','ALGORITHM','CIPHERTEXT')
) = 4, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'AUTH_SIGNUP_CONTINUATION'
      AND COLUMN_NAME IN ('KEY_ID','NONCE','ALGORITHM','CIPHERTEXT','PROVIDER_CREDENTIAL')
) = 0, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.TRIGGERS
    WHERE TRIGGER_SCHEMA = DATABASE()
      AND TRIGGER_NAME IN (
          'TRG_WEO_MEMBER_AUTH_PRINCIPAL_INSERT',
          'TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE'
      )
) = 2, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM information_schema.TRIGGERS
    WHERE TRIGGER_SCHEMA = DATABASE()
      AND EVENT_OBJECT_TABLE = 'AUTH_IDENTITY'
) = 0, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM _migration_history
    WHERE filename IN (
        '040_convert_auth_transaction_boundary_to_innodb.sql',
        '041_create_canonical_identity_schema.sql',
        '042_prepare_canonical_auth_cutover.sql'
    )
) = 3, 0, 1);

SET @canonical_contract_failures = @canonical_contract_failures + IF((
    SELECT COUNT(*)
    FROM _migration_history
    WHERE filename = '043_finalize_identity_authority.sql'
) = 0, 0, 1);

-- canonical_schema_columns_fingerprint_v1
SELECT SHA2(GROUP_CONCAT(CONCAT_WS('|',
           TABLE_NAME, LPAD(ORDINAL_POSITION, 3, '0'), COLUMN_NAME,
           COLUMN_TYPE, IS_NULLABLE, IFNULL(HEX(COLUMN_DEFAULT), '<NULL>'),
           EXTRA, IFNULL(CHARACTER_SET_NAME, '<NULL>'), IFNULL(COLLATION_NAME, '<NULL>'))
       ORDER BY TABLE_NAME, ORDINAL_POSITION SEPARATOR '||'), 256)
INTO @canonical_columns_fingerprint
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND FIND_IN_SET(TABLE_NAME, @canonical_tables);
SET @canonical_contract_failures = @canonical_contract_failures + IF(
    @canonical_columns_fingerprint = 'e2c33ce55ea20b15690d9f1f10122259e67c6b8bbe69ff38133d58a50d07d629',
    0, 1
);

-- canonical_schema_indexes_fingerprint_v1
SELECT SHA2(GROUP_CONCAT(CONCAT_WS('|',
           TABLE_NAME, INDEX_NAME, NON_UNIQUE, LPAD(SEQ_IN_INDEX, 3, '0'),
           COLUMN_NAME, IFNULL(SUB_PART, '<NULL>'), INDEX_TYPE,
           IFNULL(COLLATION, '<NULL>'), NULLABLE)
       ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX SEPARATOR '||'), 256)
INTO @canonical_indexes_fingerprint
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND FIND_IN_SET(TABLE_NAME, @canonical_tables);
SET @canonical_contract_failures = @canonical_contract_failures + IF(
    @canonical_indexes_fingerprint = 'ecc521898fec8e3ace5d04c629e99831137df6ef646b0edeb86737f58dc879f0',
    0, 1
);

-- canonical_schema_foreign_keys_fingerprint_v1
SELECT SHA2(GROUP_CONCAT(CONCAT_WS('|',
           k.TABLE_NAME, k.CONSTRAINT_NAME, LPAD(k.ORDINAL_POSITION, 3, '0'),
           k.COLUMN_NAME, k.REFERENCED_TABLE_NAME, k.REFERENCED_COLUMN_NAME,
           r.UPDATE_RULE, r.DELETE_RULE)
       ORDER BY k.TABLE_NAME, k.CONSTRAINT_NAME, k.ORDINAL_POSITION SEPARATOR '||'), 256)
INTO @canonical_foreign_keys_fingerprint
FROM information_schema.KEY_COLUMN_USAGE k
JOIN information_schema.TABLE_CONSTRAINTS t
  ON t.CONSTRAINT_SCHEMA = k.CONSTRAINT_SCHEMA
 AND t.TABLE_NAME = k.TABLE_NAME
 AND t.CONSTRAINT_NAME = k.CONSTRAINT_NAME
JOIN information_schema.REFERENTIAL_CONSTRAINTS r
  ON r.CONSTRAINT_SCHEMA = k.CONSTRAINT_SCHEMA
 AND r.TABLE_NAME = k.TABLE_NAME
 AND r.CONSTRAINT_NAME = k.CONSTRAINT_NAME
WHERE k.CONSTRAINT_SCHEMA = DATABASE()
  AND t.CONSTRAINT_TYPE = 'FOREIGN KEY'
  AND FIND_IN_SET(k.TABLE_NAME, @canonical_tables);
SET @canonical_contract_failures = @canonical_contract_failures + IF(
    @canonical_foreign_keys_fingerprint = 'be04e76e2f29beed5dcb1347fba5ca13d3594aeb54e12ff3974141ba382a810a',
    0, 1
);

-- canonical_schema_tables_fingerprint_v1
SELECT SHA2(GROUP_CONCAT(CONCAT_WS('|', TABLE_NAME, ENGINE, TABLE_COLLATION)
       ORDER BY TABLE_NAME SEPARATOR '||'), 256)
INTO @canonical_tables_fingerprint
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE()
  AND FIND_IN_SET(TABLE_NAME, @canonical_tables);
SET @canonical_contract_failures = @canonical_contract_failures + IF(
    @canonical_tables_fingerprint = '94d5dcd242671c0777b4b0055d95517a5215c09afe2c06a02062587e20f28c11',
    0, 1
);

-- canonical_schema_triggers_fingerprint_v1
SELECT SHA2(GROUP_CONCAT(CONCAT_WS('|',
           TRIGGER_NAME, EVENT_MANIPULATION, ACTION_TIMING, EVENT_OBJECT_TABLE,
           ACTION_ORIENTATION, ACTION_STATEMENT)
       ORDER BY TRIGGER_NAME SEPARATOR '||'), 256)
INTO @canonical_triggers_fingerprint
FROM information_schema.TRIGGERS
WHERE TRIGGER_SCHEMA = DATABASE()
  AND (
      FIND_IN_SET(EVENT_OBJECT_TABLE, @canonical_tables)
      OR TRIGGER_NAME IN (
          'TRG_WEO_MEMBER_AUTH_PRINCIPAL_INSERT',
          'TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE'
      )
  );
SET @canonical_contract_failures = @canonical_contract_failures + IF(
    @canonical_triggers_fingerprint = '28f734ccb2c28b96c1d078e84d51f1de1ba071c9d2e53323314d46965efd8cd8',
    0, 1
);

-- canonical_schema_history_fingerprint_v1
SELECT SHA2(GROUP_CONCAT(CONCAT_WS('|', filename, sha256)
       ORDER BY filename SEPARATOR '||'), 256)
INTO @canonical_history_fingerprint
FROM _migration_history
WHERE filename IN (
    '040_convert_auth_transaction_boundary_to_innodb.sql',
    '041_create_canonical_identity_schema.sql',
    '042_prepare_canonical_auth_cutover.sql'
);
SET @canonical_contract_failures = @canonical_contract_failures + IF(
    @canonical_history_fingerprint = '10eb9562321138d0f24729bfd317e4e5156f16561505fa9ed2d54ecd9f84a3c0',
    0, 1
);

SELECT IF(
    @canonical_contract_failures = 0,
    'CANONICAL_SCHEMA_CONTRACT_PASS',
    'CANONICAL_SCHEMA_CONTRACT_FAIL'
);
