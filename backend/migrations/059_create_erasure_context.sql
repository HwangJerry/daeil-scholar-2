-- Encrypted, request-bound external work survives operational account deletion.
-- TTL is an operational handoff limit, not a claim about backup retention.
CREATE TABLE IF NOT EXISTS ALUMNI_ERASURE_CONTEXT (
    REQUEST_ID BIGINT NOT NULL PRIMARY KEY,
    CIPHERTEXT MEDIUMBLOB NOT NULL,
    EXPIRES_AT DATETIME NOT NULL,
    CREATED_AT DATETIME NOT NULL,
    KEY idx_erasure_context_expiry (EXPIRES_AT)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
