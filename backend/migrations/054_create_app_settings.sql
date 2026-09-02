-- Migration 054: Generic remote settings for mobile applications.
-- Target: MariaDB 10.1.38.

CREATE TABLE IF NOT EXISTS app_settings (
    AS_KEY         VARCHAR(100) NOT NULL,
    AS_VALUE       TEXT NOT NULL,
    AS_DESCRIPTION VARCHAR(500) NOT NULL DEFAULT '',
    AS_PUBLIC      CHAR(1) NOT NULL DEFAULT 'N',
    UPDATED_AT     DATETIME NOT NULL,
    UPDATED_BY     INT NULL,
    PRIMARY KEY (AS_KEY),
    INDEX IDX_APP_SETTINGS_PUBLIC (AS_PUBLIC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO app_settings (
    AS_KEY, AS_VALUE, AS_DESCRIPTION, AS_PUBLIC, UPDATED_AT, UPDATED_BY
)
SELECT
    'kakao_open_chat_url',
    'https://open.kakao.com/o/gNLYTuui',
    '아이디/비밀번호 찾기 문의용 카카오톡 오픈채팅 URL',
    'Y',
    NOW(),
    NULL
FROM DUAL
WHERE NOT EXISTS (
    SELECT 1
    FROM app_settings
    WHERE AS_KEY = 'kakao_open_chat_url'
);

-- Rollback:
-- DROP TABLE app_settings;
