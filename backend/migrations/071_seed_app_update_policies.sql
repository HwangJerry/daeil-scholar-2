-- Migration 071: Per-platform app update policies stored as JSON in app_settings.
-- Target: MariaDB 10.1.38 (no JSON column type — TEXT + app-level parsing).
-- Both policies ship disabled, so applying this migration changes no behavior.

INSERT INTO app_settings (
    AS_KEY, AS_VALUE, AS_DESCRIPTION, AS_PUBLIC, UPDATED_AT, UPDATED_BY
)
SELECT
    'app_update_policy_ios',
    '{"forceEnabled":false,"minBuild":0,"recommendEnabled":false,"recommendedBuild":0,"minOsVersion":"17.0","storeUrl":""}',
    'iOS 앱 업데이트 정책. 관리자 화면에서만 수정한다',
    'Y',
    NOW(),
    NULL
FROM DUAL
WHERE NOT EXISTS (
    SELECT 1
    FROM app_settings
    WHERE AS_KEY = 'app_update_policy_ios'
);

INSERT INTO app_settings (
    AS_KEY, AS_VALUE, AS_DESCRIPTION, AS_PUBLIC, UPDATED_AT, UPDATED_BY
)
SELECT
    'app_update_policy_android',
    '{"forceEnabled":false,"minBuild":0,"recommendEnabled":false,"recommendedBuild":0,"minOsVersion":"26"}',
    'Android 앱 업데이트 정책. 관리자 화면에서만 수정한다',
    'Y',
    NOW(),
    NULL
FROM DUAL
WHERE NOT EXISTS (
    SELECT 1
    FROM app_settings
    WHERE AS_KEY = 'app_update_policy_android'
);

-- Rollback:
-- DELETE FROM app_settings WHERE AS_KEY IN ('app_update_policy_ios', 'app_update_policy_android');
