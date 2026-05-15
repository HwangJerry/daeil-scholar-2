-- Migration 025: Add manual donor count override to donation config
ALTER TABLE DONATION_CONFIG
  ADD COLUMN DC_MANUAL_DONOR_CNT INT NOT NULL DEFAULT 0;
