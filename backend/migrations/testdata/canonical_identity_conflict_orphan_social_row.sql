-- Independent orphan social-link conflict.
USE canonical_identity_test;
INSERT INTO WEO_MEMBER_SOCIAL (
    USR_SEQ, NMS_GATE, NMS_ID, NMS_EMAIL, NMS_STATUS, NMS_EMAIL_ENABLED, NMS_NAME, REG_DATE
) VALUES (
    819999, 'KAKAO', 'orphan-fixture-subject', NULL, 'ACTIVE', 'N', 'Orphan Fixture', NOW()
);
