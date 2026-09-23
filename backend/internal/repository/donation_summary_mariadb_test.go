package repository

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestDonationBalanceAndMonthAmountOnMariaDB101(t *testing.T) {
	if os.Getenv("DONATION_SUMMARY_DOCKER_INTEGRATION") != "1" {
		t.Skip("set DONATION_SUMMARY_DOCKER_INTEGRATION=1 for pinned MariaDB 10.1.38")
	}
	db := startPasswordResetMariaDB101(t)
	_, err := db.Exec(`CREATE TABLE DONATION_CONFIG (
		DC_SEQ INT PRIMARY KEY, DC_GOAL BIGINT DEFAULT 0, DC_MANUAL_ADJ BIGINT DEFAULT 0,
		DC_MANUAL_DONOR_CNT INT DEFAULT 0,
		DC_TIER_SPROUT_MIN BIGINT DEFAULT 1, DC_TIER_SAPLING_MIN BIGINT DEFAULT 10000,
		DC_TIER_TREE_MIN BIGINT DEFAULT 50000, DC_TIER_BLOOMING_MIN BIGINT DEFAULT 100000,
		DC_TIER_FRUITING_MIN BIGINT DEFAULT 300000, DC_NOTE TEXT NULL,
		DC_OVERWRITE CHAR(1) DEFAULT 'N', IS_ACTIVE CHAR(1), REG_DATE DATETIME NULL, REG_OPER INT NULL
	) ENGINE=InnoDB;
	INSERT INTO DONATION_CONFIG (DC_SEQ,IS_ACTIVE) VALUES (1,'Y'), (2,'N');
	CREATE TABLE WEO_ORDER (O_TYPE CHAR(1), O_LIFECYCLE_STATUS VARCHAR(30), O_NET_RECEIVED_AMOUNT BIGINT, O_DONATION_DATE DATE NULL);
	INSERT INTO WEO_ORDER VALUES
	('A','completed',900,'2024-01-31'),
	('A','completed',100,'2024-02-01'),
	('A','partially_refunded',300,'2024-02-29'),
	('A','completed',700,'2024-03-01'),
	('A','pending',800,'2024-02-15'),
	('A','refunded',600,'2024-02-15'),
	('B','completed',500,'2024-02-15'),
	('A','completed',400,NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/077_add_donation_account_balance.sql")
	if err != nil {
		t.Fatal(err)
	}
	// DELIMITER is a mysql CLI command; the driver sends the procedure as SQL.
	migrationSQL := strings.NewReplacer("DELIMITER //", "", "DELIMITER ;", "", "END //", "END;").Replace(string(migration))
	if _, err := db.Exec(migrationSQL); err != nil {
		t.Fatal(err)
	}
	repo := NewDonationRepository(db)
	config, err := repo.GetActiveConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.BalanceAmount != nil || config.BalanceAsOf != nil {
		t.Fatalf("existing config should have NULL balance: %+v", config)
	}
	amount, date := int64(5000000000), "2024-02-29"
	config.BalanceAmount, config.BalanceAsOf = &amount, &date
	if err := NewAdminDonationRepository(db).UpdateConfig(*config, 7); err != nil {
		t.Fatal(err)
	}
	// Reapplying must retain the saved BIGINT and DATE.
	if _, err := db.Exec(migrationSQL); err != nil {
		t.Fatal(err)
	}
	config, err = repo.GetActiveConfig()
	if err != nil || config.BalanceAmount == nil || *config.BalanceAmount != amount || config.BalanceAsOf == nil || *config.BalanceAsOf != date {
		t.Fatalf("round trip config=%+v error=%v", config, err)
	}
	var inactiveBalance *int64
	if err := db.Get(&inactiveBalance, `SELECT DC_BALANCE_AMOUNT FROM DONATION_CONFIG WHERE DC_SEQ=2`); err != nil || inactiveBalance != nil {
		t.Fatalf("inactive balance changed: %v, error=%v", inactiveBalance, err)
	}
	start := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
	total, err := repo.GetReceivedDonationAmountBetween(start, start.AddDate(0, 1, 0))
	if err != nil || total != 400 {
		t.Fatalf("February total=%d error=%v, want 400", total, err)
	}
	start = start.AddDate(0, 2, 0)
	total, err = repo.GetReceivedDonationAmountBetween(start, start.AddDate(0, 1, 0))
	if err != nil || total != 0 {
		t.Fatalf("empty month total=%d error=%v, want 0", total, err)
	}
	config.BalanceAmount, config.BalanceAsOf = nil, nil
	if err := NewAdminDonationRepository(db).UpdateConfig(*config, 7); err != nil {
		t.Fatal(err)
	}
	config, err = repo.GetActiveConfig()
	if err != nil || config.BalanceAmount != nil || config.BalanceAsOf != nil {
		t.Fatalf("cleared config=%+v error=%v", config, err)
	}
}
