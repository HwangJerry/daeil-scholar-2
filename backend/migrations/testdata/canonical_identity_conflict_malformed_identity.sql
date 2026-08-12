-- Independent malformed provider identity conflict.
USE canonical_identity_test;
INSERT INTO WEO_MEMBER_SOCIAL (
    USR_SEQ, NMS_GATE, NMS_ID, NMS_EMAIL, NMS_STATUS, NMS_EMAIL_ENABLED, NMS_NAME, REG_DATE
) VALUES
    (810001, '', 'malformed-fixture-subject', NULL, 'ACTIVE', 'N', 'Malformed Gate', NOW()),
    (810002, 'KAKAO', '', NULL, 'ACTIVE', 'N', 'Malformed Subject', NOW());
