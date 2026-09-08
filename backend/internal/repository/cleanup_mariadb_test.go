package repository

import (
	"context"
	"os"
	"testing"
	"time"
)

// This test creates a fresh local database with synthetic records only.
func TestCleanupBoundariesOnMariaDB101(t *testing.T) {
	if os.Getenv("CLEANUP_DOCKER_INTEGRATION") != "1" {
		t.Skip("set CLEANUP_DOCKER_INTEGRATION=1")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE USER_SESSION (ID INT PRIMARY KEY, EXPIRES_AT DATETIME) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_PASSWORD_RESET (ID INT PRIMARY KEY, EXPIRES_AT DATETIME) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_MOBILE_REFRESH_TOKEN (ID INT PRIMARY KEY, EXPIRES_AT DATETIME, REVOKED_AT DATETIME, CONSUMED_AT DATETIME) ENGINE=InnoDB;
 CREATE TABLE WEO_VISIT_DAILY (VD_DATE DATE, VD_VISITOR_ID VARCHAR(30), PRIMARY KEY (VD_DATE, VD_VISITOR_ID)) ENGINE=InnoDB;
 CREATE TABLE WEO_VISIT_SUMMARY (VS_DATE DATE PRIMARY KEY) ENGINE=InnoDB;`)
	tables := []string{"USER_SESSION", "ALUMNI_PASSWORD_RESET", "ALUMNI_MOBILE_REFRESH_TOKEN"}
	for _, table := range tables {
		for i := 1; i <= 105; i++ {
			db.MustExec("INSERT INTO "+table+" (ID, EXPIRES_AT) VALUES (?, DATE_SUB(NOW(), INTERVAL 1 DAY))", i)
		}
		db.MustExec("INSERT INTO " + table + " (ID, EXPIRES_AT) VALUES (200, DATE_ADD(NOW(), INTERVAL 1 DAY))")
	}
	cleanups := []func() (int64, error){NewSessionRepository(db).DeleteExpiredSessions, NewPasswordResetRepository(db).DeleteExpiredTokens, func() (int64, error) {
		return NewAuthRepository(db).DeleteExpiredMobileRefreshTokens(time.Now().AddDate(0, 0, -7))
	}}
	for _, cleanup := range cleanups {
		if n, err := cleanup(); err != nil || n != 100 {
			t.Fatalf("batch = %d, error %v", n, err)
		}
	}
	for _, table := range tables {
		var n int
		if err := db.Get(&n, "SELECT COUNT(*) FROM "+table); err != nil || n != 6 {
			t.Fatalf("%s remaining=%d: %v", table, n, err)
		}
	}
	db.MustExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN SET CONSUMED_AT=NOW() WHERE ID=200;
 INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN VALUES
 (201,DATE_ADD(NOW(),INTERVAL 1 DAY),DATE_SUB(NOW(),INTERVAL 8 DAY),NULL),
 (202,DATE_ADD(NOW(),INTERVAL 1 DAY),NOW(),NULL);`)
	if n, err := cleanups[2](); err != nil || n != 6 {
		t.Fatalf("revoked cleanup = %d, %v", n, err)
	}
	var n int
	if err := db.Get(&n, "SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE ID IN (200,202)"); err != nil || n != 2 {
		t.Fatalf("valid/recent tokens lost: %d %v", n, err)
	}
	db.MustExec(`INSERT INTO WEO_VISIT_SUMMARY VALUES ('2026-01-01');
 INSERT INTO WEO_VISIT_DAILY VALUES ('2026-01-02','missing-summary'),('2026-08-01','recent');`)
	for i := 1; i <= 105; i++ {
		db.MustExec("INSERT INTO WEO_VISIT_DAILY VALUES ('2026-01-01',?)", i)
	}
	visits := NewVisitRepository(db)
	cutoff := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if n, err := visits.DeleteDailyBefore(cutoff); err != nil || n != 100 {
		t.Fatalf("visits batch=%d: %v", n, err)
	}
	if n, err := visits.DeleteDailyBefore(cutoff); err != nil || n != 5 {
		t.Fatalf("visits remainder=%d: %v", n, err)
	}
	if err := db.Get(&n, "SELECT COUNT(*) FROM WEO_VISIT_DAILY"); err != nil || n != 2 {
		t.Fatalf("unaggregated/recent visits lost: %d %v", n, err)
	}
	if err := db.Get(&n, "SELECT COUNT(*) FROM WEO_VISIT_SUMMARY"); err != nil || n != 1 {
		t.Fatalf("summary lost: %d %v", n, err)
	}
	db.MustExec(`CREATE TABLE ALUMNI_MESSAGE_REPORT (STATUS VARCHAR(20), RESOLVED_AT DATETIME) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_DONATION_LEGAL_ARCHIVE (RETAIN_UNTIL DATE) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_ACCOUNT_DELETION_REQUEST (REQUEST_ID BIGINT PRIMARY KEY, STATUS VARCHAR(20), COMPLETED_AT DATETIME) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_ACCOUNT_ERASURE (REQUEST_ID BIGINT PRIMARY KEY) ENGINE=InnoDB;
 INSERT INTO ALUMNI_ACCOUNT_DELETION_REQUEST VALUES (1,'completed',DATE_SUB(NOW(),INTERVAL 40 DAY)),(2,'completed',DATE_SUB(NOW(),INTERVAL 50 DAY)),(3,'pending',NULL);
 INSERT INTO ALUMNI_ACCOUNT_ERASURE VALUES (1),(2),(3);`)
	retention := &AccountDeletionRequestRepository{DB: db}
	if err := retention.PurgeExpiredPrivacyRecords(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&n, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_ERASURE e LEFT JOIN ALUMNI_ACCOUNT_DELETION_REQUEST d ON d.REQUEST_ID=e.REQUEST_ID WHERE d.REQUEST_ID IS NULL`); err != nil || n != 0 {
		t.Fatalf("orphan erasure state: %d %v", n, err)
	}
	if err := retention.PurgeExpiredPrivacyRecords(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&n, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE STATUS='completed'`); err != nil || n != 0 {
		t.Fatalf("completed receipts not drained: %d %v", n, err)
	}
	if err := db.Get(&n, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=3`); err != nil || n != 1 {
		t.Fatalf("pending receipt lost: %d %v", n, err)
	}

}
