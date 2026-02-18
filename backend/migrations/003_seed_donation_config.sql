-- Migration 003: Seed initial donation config
-- Insert a default active configuration row so the snapshot job has data to work with

INSERT INTO DONATION_CONFIG (DC_GOAL, DC_MANUAL_ADJ, DC_NOTE, IS_ACTIVE, REG_DATE)
VALUES (200000000, 0, '초기 설정', 'Y', NOW());
