-- Migration 081: Account-wide message receiving, independent of push alerts.
-- Existing members remain reachable. Target: MariaDB 10.1.38; safe to rerun.
ALTER TABLE WEO_MEMBER ADD COLUMN IF NOT EXISTS USR_MESSAGE_ALLOWED CHAR(1) NOT NULL DEFAULT 'Y';
-- Rollback: ALTER TABLE WEO_MEMBER DROP COLUMN USR_MESSAGE_ALLOWED;
