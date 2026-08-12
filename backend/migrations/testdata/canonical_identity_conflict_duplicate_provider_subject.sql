-- Independent provider-subject ownership conflict for immutable migration 037.
INSERT INTO WEO_MEMBER_SOCIAL (
    NMS_GATE, USR_SEQ, NMS_ID, NMS_EMAIL, NMS_STATUS, REG_DATE
) VALUES (
    'KT', 102, 'fixture-subject-1', 'conflict@example.invalid', 'Y', NOW()
);
