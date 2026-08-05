-- Read-only Kakao auth migration preflight. Every metric must be zero.
SELECT 'missing_required_tables' AS metric,
       3 - COUNT(*) AS violations
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME IN ('WEO_MEMBER', 'WEO_MEMBER_SOCIAL', 'ALUMNI_MOBILE_REFRESH_TOKEN')
UNION ALL
SELECT 'duplicate_user_provider_groups', COUNT(*)
FROM (
    SELECT 1
    FROM WEO_MEMBER_SOCIAL
    GROUP BY USR_SEQ, NMS_GATE
    HAVING COUNT(*) > 1
) AS duplicate_user_provider
UNION ALL
SELECT 'conflicting_provider_subject_groups', COUNT(*)
FROM (
    SELECT 1
    FROM WEO_MEMBER_SOCIAL
    GROUP BY NMS_GATE, NMS_ID
    HAVING COUNT(DISTINCT USR_SEQ) > 1
) AS conflicting_provider_subject
UNION ALL
SELECT 'unsupported_social_status_rows', COUNT(*)
FROM WEO_MEMBER_SOCIAL
WHERE NMS_STATUS IS NULL
   OR NMS_STATUS NOT IN ('Y', 'N', 'ACTIVE', 'INACTIVE')
UNION ALL
SELECT 'cohort_source_overflow_rows', COUNT(*)
FROM WEO_MEMBER
WHERE CHAR_LENGTH(TRIM(COALESCE(USR_FN, ''))) > 20
UNION ALL
SELECT 'department_source_overflow_rows', COUNT(*)
FROM WEO_MEMBER
WHERE CHAR_LENGTH(TRIM(COALESCE(USR_DEPT, ''))) > 100;
