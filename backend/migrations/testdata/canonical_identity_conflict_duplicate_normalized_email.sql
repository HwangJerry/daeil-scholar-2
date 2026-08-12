-- Independent normalized-email conflict; no credential values are present.
USE canonical_identity_test;
INSERT INTO WEO_MEMBER (
    USR_SEQ, USR_ID, USR_EMAIL, USR_PWD, USR_NAME, USR_PHONE, USR_FN, USR_DEPT, USR_STATUS
) VALUES
    (810001, 'email-conflict-a', 'duplicate@example.invalid', NULL, 'Fixture A', '01000000001', '0', 'Fixture', 'AAA'),
    (810002, 'email-conflict-b', ' DUPLICATE@EXAMPLE.INVALID ', NULL, 'Fixture B', '01000000002', '0', 'Fixture', 'AAA');
