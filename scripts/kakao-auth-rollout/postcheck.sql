-- Read-only Kakao auth migration postcheck. Every metric must be zero.
SELECT 'social_engine_not_innodb' AS metric,
       IF(COALESCE(MAX(ENGINE), '') = 'InnoDB', 0, 1) AS violations
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'WEO_MEMBER_SOCIAL'
UNION ALL
SELECT 'missing_canonical_social_columns',
       3 - COUNT(*)
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'WEO_MEMBER_SOCIAL'
  AND COLUMN_NAME IN ('NMS_EMAIL', 'NMS_STATUS', 'NMS_EMAIL_ENABLED')
UNION ALL
SELECT 'invalid_social_status_column_shape',
       IF(COUNT(*) = 1, 0, 1)
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'WEO_MEMBER_SOCIAL'
  AND COLUMN_NAME = 'NMS_STATUS'
  AND COLUMN_TYPE = 'varchar(20)'
  AND IS_NULLABLE = 'NO'
  AND COLUMN_DEFAULT = 'ACTIVE'
UNION ALL
SELECT 'invalid_social_email_column_shape',
       IF(COUNT(*) = 1, 0, 1)
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'WEO_MEMBER_SOCIAL'
  AND COLUMN_NAME = 'NMS_EMAIL'
  AND COLUMN_TYPE = 'varchar(255)'
  AND IS_NULLABLE = 'YES'
UNION ALL
SELECT 'legacy_active_status_rows', COUNT(*)
FROM WEO_MEMBER_SOCIAL
WHERE NMS_STATUS = 'Y'
UNION ALL
SELECT 'duplicate_user_provider_groups', COUNT(*)
FROM (
    SELECT 1 FROM WEO_MEMBER_SOCIAL
    GROUP BY USR_SEQ, NMS_GATE HAVING COUNT(*) > 1
) AS duplicate_user_provider
UNION ALL
SELECT 'conflicting_provider_subject_groups', COUNT(*)
FROM (
    SELECT 1 FROM WEO_MEMBER_SOCIAL
    GROUP BY NMS_GATE, NMS_ID HAVING COUNT(DISTINCT USR_SEQ) > 1
) AS conflicting_provider_subject
UNION ALL
SELECT 'missing_unique_user_provider_index',
       IF(COUNT(*) > 0, 0, 1)
FROM (
    SELECT INDEX_NAME
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'WEO_MEMBER_SOCIAL'
      AND NON_UNIQUE = 0
    GROUP BY INDEX_NAME
    HAVING GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) = 'USR_SEQ,NMS_GATE'
) AS user_provider_index
UNION ALL
SELECT 'missing_unique_provider_subject_index',
       IF(COUNT(*) > 0, 0, 1)
FROM (
    SELECT INDEX_NAME
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'WEO_MEMBER_SOCIAL'
      AND NON_UNIQUE = 0
    GROUP BY INDEX_NAME
    HAVING GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) = 'NMS_GATE,NMS_ID'
) AS provider_subject_index
UNION ALL
SELECT 'missing_refresh_rotation_columns',
       3 - COUNT(*)
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'ALUMNI_MOBILE_REFRESH_TOKEN'
  AND COLUMN_NAME IN ('CONSUMED_AT', 'REVOKED_AT', 'ROTATED_TO_JTI')
UNION ALL
SELECT 'missing_refresh_sid_index',
       IF(COUNT(*) > 0, 0, 1)
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'ALUMNI_MOBILE_REFRESH_TOKEN'
  AND INDEX_NAME = 'IDX_MRT_SID'
UNION ALL
SELECT 'missing_auth_tables',
       4 - COUNT(*)
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME IN (
      'ALUMNI_ADMIN_ROLE',
      'ALUMNI_VERIFICATION',
      'ALUMNI_SOCIAL_LINK_REAUTH_GUARD',
      'ALUMNI_SOCIAL_LINK_CONTINUATION'
  )
UNION ALL
SELECT 'invalid_auth_table_engines', COUNT(*)
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME IN (
      'ALUMNI_ADMIN_ROLE',
      'ALUMNI_VERIFICATION',
      'ALUMNI_SOCIAL_LINK_REAUTH_GUARD',
      'ALUMNI_SOCIAL_LINK_CONTINUATION'
  )
  AND ENGINE <> 'InnoDB'
UNION ALL
SELECT 'legacy_verification_projection_mismatch', COUNT(*)
FROM WEO_MEMBER AS m
LEFT JOIN ALUMNI_VERIFICATION AS v ON v.USR_SEQ = m.USR_SEQ
WHERE m.USR_STATUS IN ('BAA', 'BBB', 'CCC', 'ZZZ')
  AND (
      v.USR_SEQ IS NULL
      OR v.STATUS <> CASE
          WHEN m.USR_STATUS = 'BAA' THEN 'rejected'
          WHEN m.USR_STATUS = 'BBB' THEN 'pending'
          ELSE 'approved'
      END
  )
UNION ALL
SELECT 'legacy_root_projection_missing', COUNT(*)
FROM WEO_MEMBER AS m
LEFT JOIN ALUMNI_ADMIN_ROLE AS r
  ON r.USR_SEQ = m.USR_SEQ AND r.ADMIN_ROLE = 'root'
WHERE m.USR_STATUS = 'ZZZ' AND r.USR_SEQ IS NULL;
