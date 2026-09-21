-- Migration 075: Admin-editable notification texts for SMS and push messages.
-- The key set is fixed in application code (internal/model/notification_template.go);
-- this table only stores the administrator's overrides of the title and body.
-- Seeds are idempotent so re-running the migration never overwrites an edit.
-- Target: MariaDB 10.1.38.

CREATE TABLE IF NOT EXISTS notification_templates (
    NT_KEY     VARCHAR(100) NOT NULL,
    NT_CHANNEL VARCHAR(10) NOT NULL,
    NT_TITLE   VARCHAR(200) NOT NULL DEFAULT '',
    NT_BODY    TEXT NOT NULL,
    NT_VERSION INT NOT NULL DEFAULT 1,
    UPDATED_AT DATETIME NOT NULL,
    UPDATED_BY INT NULL,
    PRIMARY KEY (NT_KEY)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO notification_templates (NT_KEY, NT_CHANNEL, NT_TITLE, NT_BODY, NT_VERSION, UPDATED_AT, UPDATED_BY)
SELECT 'sms.phone_verification', 'sms', '', '[대일외고장학회] 인증번호 {code} 를 입력해 주세요.', 1, NOW(), NULL
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM notification_templates WHERE NT_KEY = 'sms.phone_verification');

INSERT INTO notification_templates (NT_KEY, NT_CHANNEL, NT_TITLE, NT_BODY, NT_VERSION, UPDATED_AT, UPDATED_BY)
SELECT 'push.verification.approved', 'push', '동문 인증 결과', '동문 인증이 승인되었습니다. 이제 동문 커뮤니티를 이용할 수 있어요.', 1, NOW(), NULL
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM notification_templates WHERE NT_KEY = 'push.verification.approved');

INSERT INTO notification_templates (NT_KEY, NT_CHANNEL, NT_TITLE, NT_BODY, NT_VERSION, UPDATED_AT, UPDATED_BY)
SELECT 'push.verification.rejected', 'push', '동문 인증 결과', '동문 인증 신청이 반려되었습니다. 앱에서 사유를 확인하고 다시 신청해 주세요.', 1, NOW(), NULL
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM notification_templates WHERE NT_KEY = 'push.verification.rejected');

INSERT INTO notification_templates (NT_KEY, NT_CHANNEL, NT_TITLE, NT_BODY, NT_VERSION, UPDATED_AT, UPDATED_BY)
SELECT 'push.message.new_preview_off', 'push', '{senderName}', '새 메시지가 도착했습니다.', 1, NOW(), NULL
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM notification_templates WHERE NT_KEY = 'push.message.new_preview_off');

INSERT INTO notification_templates (NT_KEY, NT_CHANNEL, NT_TITLE, NT_BODY, NT_VERSION, UPDATED_AT, UPDATED_BY)
SELECT 'push.message.new_preview_on', 'push', '{senderName}', '{content}', 1, NOW(), NULL
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM notification_templates WHERE NT_KEY = 'push.message.new_preview_on');

INSERT INTO notification_templates (NT_KEY, NT_CHANNEL, NT_TITLE, NT_BODY, NT_VERSION, UPDATED_AT, UPDATED_BY)
SELECT 'push.notice.new', 'push', '새 소식', '{subject}', 1, NOW(), NULL
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM notification_templates WHERE NT_KEY = 'push.notice.new');

-- Rollback:
-- DROP TABLE notification_templates;
