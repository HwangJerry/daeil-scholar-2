-- Independent legacy-password algorithm conflict; the literal is a non-secret marker.
USE canonical_identity_test;
INSERT INTO WEO_MEMBER (
    USR_SEQ, USR_ID, USR_EMAIL, USR_PWD, USR_NAME, USR_PHONE, USR_FN, USR_DEPT, USR_STATUS
) VALUES (
    810001, 'algorithm-conflict', NULL, 'UNREADABLE_ALGORITHM_TAG:fixture', 'Fixture', '01000000001', '0', 'Fixture', 'AAA'
);
