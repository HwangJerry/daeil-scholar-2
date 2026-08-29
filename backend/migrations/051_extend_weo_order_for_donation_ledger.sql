-- Migration 051: Extend WEO_ORDER into the canonical private donation ledger.
-- Target: MariaDB 10.1.38. Apply manually after a backup.
-- New columns are additive so current readers of legacy columns remain valid.
-- Preflight is intentionally fail-closed. Legacy N rows cannot distinguish a
-- cancellation from a full refund and must be classified before this migration.
--
-- Formalized from backend/migrations/testdata/current-branch-8dcba0b/033_extend_weo_order_for_donation_ledger.sql,
-- which was a branch-lineage test fixture, not a numbered migration ever applied via migrate.sh.
-- The application code that depends on these columns (DonationRepository.GetReceivedDonationAggregate)
-- had already been deployed to production without this migration ever running, causing
-- /api/donation/summary to fail with "Unknown column 'O_NET_RECEIVED_AMOUNT'". Applied to
-- production on 2026-08-29 after a WEO_ORDER backup and a clean preflight run (0/5257 rows failed).

DROP TEMPORARY TABLE IF EXISTS _MVP_DONATION_PREFLIGHT_ERRORS;
CREATE TEMPORARY TABLE _MVP_DONATION_PREFLIGHT_ERRORS AS
SELECT O_SEQ, 'ambiguous_cancel_or_refund' AS ERROR_CODE
FROM WEO_ORDER
WHERE O_STATUS = 'N'
UNION ALL
SELECT O_SEQ, 'payment_yes_status_not_completed'
FROM WEO_ORDER
WHERE O_PAYMENT = 'Y' AND (O_STATUS IS NULL OR O_STATUS <> 'Y')
UNION ALL
SELECT O_SEQ, 'completed_status_payment_not_yes'
FROM WEO_ORDER
WHERE O_STATUS = 'Y' AND (O_PAYMENT IS NULL OR O_PAYMENT <> 'Y')
UNION ALL
SELECT O_SEQ, 'negative_price'
FROM WEO_ORDER
WHERE O_PRICE < 0
UNION ALL
SELECT O_SEQ, 'negative_paid_amount'
FROM WEO_ORDER
WHERE O_PAY < 0
UNION ALL
SELECT O_SEQ, 'missing_price'
FROM WEO_ORDER
WHERE O_PRICE IS NULL
UNION ALL
SELECT O_SEQ, 'unknown_legacy_status'
FROM WEO_ORDER
WHERE COALESCE(O_STATUS, '') NOT IN ('I','Y','N')
UNION ALL
SELECT O_SEQ, 'unknown_payment_flag'
FROM WEO_ORDER
WHERE COALESCE(O_PAYMENT, '') NOT IN ('Y','N')
UNION ALL
SELECT O_SEQ, 'completed_price_paid_mismatch'
FROM WEO_ORDER
WHERE O_PAYMENT = 'Y' AND O_STATUS = 'Y'
  AND (O_PRICE IS NULL OR O_PAY IS NULL OR O_PRICE <> O_PAY)
UNION ALL
SELECT O_SEQ, 'missing_donation_date'
FROM WEO_ORDER
WHERE COALESCE(O_PAYDATE, O_REGDATE) IS NULL;

SELECT ERROR_CODE, COUNT(*) AS ERROR_COUNT
FROM _MVP_DONATION_PREFLIGHT_ERRORS
GROUP BY ERROR_CODE
ORDER BY ERROR_CODE;

DROP TEMPORARY TABLE IF EXISTS _MVP_DONATION_PREFLIGHT_GUARD;
CREATE TEMPORARY TABLE _MVP_DONATION_PREFLIGHT_GUARD (
    GUARD_ID TINYINT NOT NULL PRIMARY KEY
);
INSERT INTO _MVP_DONATION_PREFLIGHT_GUARD (GUARD_ID) VALUES (1);
-- Any preflight error deliberately raises duplicate-key error 1062 before DDL.
INSERT INTO _MVP_DONATION_PREFLIGHT_GUARD (GUARD_ID)
SELECT 1 FROM _MVP_DONATION_PREFLIGHT_ERRORS LIMIT 1;
DROP TEMPORARY TABLE _MVP_DONATION_PREFLIGHT_GUARD;
DROP TEMPORARY TABLE _MVP_DONATION_PREFLIGHT_ERRORS;

ALTER TABLE WEO_ORDER
    ADD COLUMN O_SOURCE VARCHAR(30) NOT NULL DEFAULT 'other',
    ADD COLUMN O_TRANSACTION_NO VARCHAR(191) CHARACTER SET ascii COLLATE ascii_bin NULL,
    ADD COLUMN O_COMPOSITE_KEY CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    ADD COLUMN O_DONATION_DATE DATE NULL,
    ADD COLUMN O_GROSS_AMOUNT BIGINT UNSIGNED NULL,
    ADD COLUMN O_REFUNDED_AMOUNT BIGINT UNSIGNED NOT NULL DEFAULT 0,
    ADD COLUMN O_NET_RECEIVED_AMOUNT BIGINT UNSIGNED NULL,
    ADD COLUMN O_LIFECYCLE_STATUS VARCHAR(30) NOT NULL DEFAULT 'pending',
    ADD COLUMN O_PAYMENT_METHOD VARCHAR(30) NULL;

UPDATE WEO_ORDER
SET
    O_DONATION_DATE = DATE(COALESCE(O_PAYDATE, O_REGDATE)),
    O_GROSS_AMOUNT = O_PRICE,
    O_REFUNDED_AMOUNT = 0,
    O_NET_RECEIVED_AMOUNT = CASE
        WHEN O_PAYMENT = 'Y' AND O_STATUS = 'Y' THEN O_PAY
        ELSE 0
    END,
    O_LIFECYCLE_STATUS = CASE
        WHEN O_PAYMENT = 'Y' AND O_STATUS = 'Y' THEN 'completed'
        ELSE 'pending'
    END,
    O_PAYMENT_METHOD = CASE UPPER(COALESCE(O_PAY_TYPE, ''))
        WHEN 'CARD' THEN 'card'
        WHEN 'BANK' THEN 'bank'
        WHEN 'VBANK' THEN 'virtual_bank'
        WHEN 'HP' THEN 'mobile'
        WHEN 'ADMS' THEN 'admin'
        ELSE 'other'
    END
WHERE O_GROSS_AMOUNT IS NULL
   OR O_NET_RECEIVED_AMOUNT IS NULL
   OR O_DONATION_DATE IS NULL;

ALTER TABLE WEO_ORDER
    ADD UNIQUE KEY UK_WO_SOURCE_TRANSACTION (O_SOURCE, O_TRANSACTION_NO),
    ADD UNIQUE KEY UK_WO_COMPOSITE_KEY (O_COMPOSITE_KEY),
    ADD INDEX IDX_WO_LIFECYCLE_DATE (O_LIFECYCLE_STATUS, O_DONATION_DATE, O_SEQ);
