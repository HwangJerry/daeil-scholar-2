-- Edge-case rows for fail-closed migration tests only.
-- Apply to a fresh production-shape fixture before migration 037.
INSERT INTO WEO_MEMBER_SOCIAL (
    NMS_GATE, USR_SEQ, NMS_ID, NMS_EMAIL, NMS_STATUS, REG_DATE
) VALUES
    ('KT', 103, 'duplicate-user-provider', 'duplicate@example.invalid', 'Y', NOW());
