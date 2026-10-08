-- Security-only login dataset; intentionally no password, token or account identifier.
CREATE TABLE IF NOT EXISTS LOGIN_SECURITY_EVENTS (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 event_id CHAR(32) NOT NULL,
 occurred_at DATETIME(6) NOT NULL,
 provider VARCHAR(16) NOT NULL,
 outcome VARCHAR(24) NOT NULL,
 response_status SMALLINT NOT NULL,
 duration_ms DOUBLE NOT NULL,
 client_ip VARCHAR(45) NOT NULL DEFAULT '',
 ip_source VARCHAR(24) NOT NULL,
 install_id VARCHAR(36) NOT NULL DEFAULT '',
 platform VARCHAR(16) NOT NULL,
 app_version VARCHAR(32) NOT NULL DEFAULT '',
 app_build VARCHAR(20) NOT NULL DEFAULT '',
 trace_id CHAR(32) NOT NULL,
 schema_version SMALLINT NOT NULL DEFAULT 1,
 UNIQUE KEY uq_login_event (event_id),
 KEY ix_login_time (occurred_at),
 KEY ix_login_outcome_time (outcome, occurred_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
