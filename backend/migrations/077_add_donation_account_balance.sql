-- Migration 077: Optional manually entered account balance and as-of date.
-- Target: MariaDB 10.1.38. Existing configs retain NULL balances.

DELIMITER //
CREATE PROCEDURE IF NOT EXISTS _077_add_donation_account_balance_if_missing()
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'DONATION_CONFIG'
          AND COLUMN_NAME = 'DC_BALANCE_AMOUNT'
    ) THEN
        ALTER TABLE DONATION_CONFIG ADD COLUMN DC_BALANCE_AMOUNT BIGINT NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'DONATION_CONFIG'
          AND COLUMN_NAME = 'DC_BALANCE_AS_OF'
    ) THEN
        ALTER TABLE DONATION_CONFIG ADD COLUMN DC_BALANCE_AS_OF DATE NULL;
    END IF;
END //
DELIMITER ;

CALL _077_add_donation_account_balance_if_missing();
DROP PROCEDURE IF EXISTS _077_add_donation_account_balance_if_missing;

-- Rollback:
-- ALTER TABLE DONATION_CONFIG DROP COLUMN DC_BALANCE_AS_OF, DROP COLUMN DC_BALANCE_AMOUNT;
