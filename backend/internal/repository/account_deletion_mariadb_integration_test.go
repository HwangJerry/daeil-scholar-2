package repository

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestPhoneClaimWriteLockSerializesMigrationStartOnMariaDB101(t *testing.T) {
	if os.Getenv("PASSWORD_RESET_DOCKER_INTEGRATION") != "1" {
		t.Skip("set PASSWORD_RESET_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}
	db := startPasswordResetMariaDB101(t)
	if _, err := db.Exec(`
		CREATE TABLE _migration_journal (
			filename VARCHAR(255) NOT NULL PRIMARY KEY,
			sha256 CHAR(64) NOT NULL,
			state VARCHAR(20) NOT NULL,
			started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME NULL
		) ENGINE=InnoDB
	`); err != nil {
		t.Fatal(err)
	}

	writeTx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	ready, err := detectPhoneClaimsInWriteTransaction(writeTx)
	if err != nil || ready {
		t.Fatalf("ready = %v, err = %v", ready, err)
	}

	migrationStarted := make(chan error, 1)
	go func() {
		_, insertErr := db.Exec(`
			INSERT INTO _migration_journal (filename, sha256, state)
			VALUES ('044_enforce_account_lifecycle_invariants.sql', REPEAT('a', 64), 'STARTED')
		`)
		migrationStarted <- insertErr
	}()
	select {
	case err := <-migrationStarted:
		t.Fatalf("migration STARTED insert bypassed write gap lock: %v", err)
	case <-time.After(250 * time.Millisecond):
	}
	if err := writeTx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-migrationStarted; err != nil {
		t.Fatal(err)
	}

	blockedTx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer blockedTx.Rollback()
	_, err = detectPhoneClaimsInWriteTransaction(blockedTx)
	if !errors.Is(err, ErrPhoneClaimsMigrating) {
		t.Fatalf("error = %v, want ErrPhoneClaimsMigrating", err)
	}
}

func TestAccountDeletionIsAtomicOnMariaDB101(t *testing.T) {
	if os.Getenv("PASSWORD_RESET_DOCKER_INTEGRATION") != "1" {
		t.Skip("set PASSWORD_RESET_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	db := startPasswordResetMariaDB101(t)
	createAccountDeletionIntegrationSchema(t, db)
	repo := NewAuthRepository(db)
	repo.EnablePhoneClaims()
	if _, err := db.Exec(`
		CREATE TRIGGER FAIL_ACCOUNT_DELETION BEFORE UPDATE ON AUTH_IDENTITY
		FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'injected deletion failure'
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AnonymizeAccountForDeletion(42); err == nil {
		t.Fatal("injected late failure must roll back account deletion")
	}
	var rollbackState struct {
		Status       string `db:"USR_STATUS"`
		OrderAccount int    `db:"USR_SEQ"`
		SocialCount  int    `db:"SOCIAL_COUNT"`
	}
	if err := db.Get(&rollbackState, `
		SELECT member.USR_STATUS, donation.USR_SEQ,
		       (SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ = 42) AS SOCIAL_COUNT
		FROM WEO_MEMBER member JOIN WEO_ORDER donation ON donation.O_SEQ = 1
		WHERE member.USR_SEQ = 42
	`); err != nil {
		t.Fatal(err)
	}
	if rollbackState.Status != "CCC" || rollbackState.OrderAccount != 42 || rollbackState.SocialCount != 1 {
		t.Fatalf("rollback state = %#v", rollbackState)
	}
	if _, err := db.Exec(`DROP TRIGGER FAIL_ACCOUNT_DELETION`); err != nil {
		t.Fatal(err)
	}

	providers, err := repo.AnonymizeAccountForDeletion(42)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 1 || providers[0] != "KT" {
		t.Fatalf("providers = %#v", providers)
	}

	var member struct {
		Status string `db:"USR_STATUS"`
		Name   string `db:"USR_NAME"`
		Phone  string `db:"USR_PHONE"`
	}
	if err := db.Get(&member, `SELECT USR_STATUS, USR_NAME, USR_PHONE FROM WEO_MEMBER WHERE USR_SEQ = 42`); err != nil {
		t.Fatal(err)
	}
	if member.Status != "AAA" || member.Name != "탈퇴한 회원" || member.Phone != "" {
		t.Fatalf("member = %#v", member)
	}
	for query, want := range map[string]int{
		`SELECT COUNT(*) FROM AUTH_PHONE_CLAIM WHERE ACCOUNT_ID = 42`:           0,
		`SELECT COUNT(*) FROM ALUMNI_VERIFICATION WHERE USR_SEQ = 42`:           0,
		`SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ = 42`:             0,
		`SELECT COUNT(*) FROM ALUMNI_SOCIAL_CREDENTIAL WHERE USR_SEQ = 42`:      1,
		`SELECT COUNT(*) FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX WHERE USR_SEQ=42`: 1,
	} {
		var count int
		if err := db.Get(&count, query); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("%s = %d, want %d", query, count, want)
		}
	}
	var order struct {
		LegacyAccount int    `db:"USR_SEQ"`
		Account       *int   `db:"O_ACCOUNT_USR_SEQ"`
		DonorName     string `db:"O_DONOR_NAME"`
		DonorPhone    string `db:"O_DONOR_PHONE"`
	}
	if err := db.Get(&order, `SELECT USR_SEQ, O_ACCOUNT_USR_SEQ, O_DONOR_NAME, O_DONOR_PHONE FROM WEO_ORDER`); err != nil {
		t.Fatal(err)
	}
	if order.LegacyAccount != 0 || order.Account != nil || order.DonorName != "홍길동" || order.DonorPhone != "01012345678" {
		t.Fatalf("order = %#v", order)
	}
	var anonymizedClientID *string
	if err := db.Get(&anonymizedClientID, `SELECT AM_CLIENT_MESSAGE_ID FROM ALUMNI_MESSAGE WHERE AM_SEQ = 2`); err != nil {
		t.Fatal(err)
	}
	if anonymizedClientID != nil {
		t.Fatalf("anonymized client ID = %q", *anonymizedClientID)
	}
}

func createAccountDeletionIntegrationSchema(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE WEO_MEMBER (
			USR_SEQ INT PRIMARY KEY, USR_ID VARCHAR(64) NOT NULL, USR_NAME VARCHAR(100) NOT NULL,
			USR_PHONE VARCHAR(32) NOT NULL, USR_EMAIL VARCHAR(255) NOT NULL, USR_PWD VARCHAR(255) NOT NULL,
			USR_NICK VARCHAR(100) NOT NULL, USR_PHOTO VARCHAR(255) NULL, USR_FN VARCHAR(20) NOT NULL,
			USR_DEPT VARCHAR(100) NOT NULL, USR_JOB_CAT INT NULL, USR_BIZ_NAME VARCHAR(100) NOT NULL,
			USR_BIZ_DESC VARCHAR(255) NOT NULL, USR_BIZ_ADDR VARCHAR(255) NOT NULL, USR_POSITION VARCHAR(100) NOT NULL,
			USR_PHONE_PUBLIC CHAR(1) NOT NULL, USR_EMAIL_PUBLIC CHAR(1) NOT NULL, USR_STATUS CHAR(3) NOT NULL,
			USR_ANONYMIZED_AT DATETIME NULL, USR_PURGE_AT DATETIME NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
		CREATE TABLE WEO_ORDER (
			O_SEQ INT PRIMARY KEY, USR_SEQ INT NOT NULL, O_ACCOUNT_USR_SEQ INT NULL,
			O_DONOR_NAME VARCHAR(100) NULL, O_DONOR_PHONE VARCHAR(11) NULL, O_DONOR_COHORT VARCHAR(20) NULL,
			O_DONOR_DEPARTMENT VARCHAR(100) NULL, O_LEGAL_RETENTION_UNTIL DATETIME NULL,
			O_ACCOUNT_UNLINKED_AT DATETIME NULL, O_DONATION_DATE DATE NULL, O_REGDATE DATETIME NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
		CREATE TABLE ALUMNI_MESSAGE (
			AM_SEQ INT PRIMARY KEY, AM_SENDER_SEQ INT NOT NULL, AM_RECVR_SEQ INT NOT NULL,
			AM_CLIENT_MESSAGE_ID VARCHAR(64) NULL,
			AM_SENDER_ACCOUNT_SEQ INT NULL, AM_RECVR_ACCOUNT_SEQ INT NULL,
			AM_SENDER_ANONYMIZED_YN CHAR(1) NOT NULL, AM_RECVR_ANONYMIZED_YN CHAR(1) NOT NULL,
			UNIQUE KEY UK_AM_SENDER_CLIENT (AM_SENDER_SEQ, AM_CLIENT_MESSAGE_ID)
		) ENGINE=InnoDB;
		CREATE TABLE AUTH_SESSION_FAMILY (ACCOUNT_ID INT, STATUS VARCHAR(16), REVOKED_AT DATETIME NULL, REVOKE_REASON_CODE VARCHAR(64) NULL, UPDATED_AT DATETIME) ENGINE=InnoDB;
		CREATE TABLE ALUMNI_MOBILE_REFRESH_TOKEN (USR_SEQ INT, REVOKED_AT DATETIME NULL) ENGINE=InnoDB;
		CREATE TABLE WEO_MEMBER_LOG (USR_SEQ INT) ENGINE=InnoDB;
		CREATE TABLE ALUMNI_PUSH_DEVICE (USR_SEQ INT) ENGINE=InnoDB;
		CREATE TABLE ALUMNI_PUSH_PREFERENCE (USR_SEQ INT) ENGINE=InnoDB;
		CREATE TABLE ALUMNI_MEMBER_BLOCK (BLOCKER_USR_SEQ INT, BLOCKED_USR_SEQ INT) ENGINE=InnoDB;
		CREATE TABLE ALUMNI_USER_TAG (USR_SEQ INT) ENGINE=InnoDB;
		CREATE TABLE ALUMNI_ADMIN_ROLE (USR_SEQ INT) ENGINE=InnoDB;
		CREATE TABLE ALUMNI_VERIFICATION (USR_SEQ INT) ENGINE=InnoDB;
		CREATE TABLE AUTH_IDENTITY (IDENTITY_ID BIGINT PRIMARY KEY, ACCOUNT_ID INT, SUBJECT_KEY VARCHAR(255), NORMALIZED_EMAIL VARCHAR(255) NULL, STATUS VARCHAR(16), REVOKED_AT DATETIME NULL, UPDATED_AT DATETIME) ENGINE=InnoDB;
		CREATE TABLE AUTH_PASSWORD_CREDENTIAL (IDENTITY_ID BIGINT PRIMARY KEY) ENGINE=InnoDB;
		CREATE TABLE AUTH_ACCOUNT_STATE (ACCOUNT_ID INT PRIMARY KEY, STATUS VARCHAR(16), WITHDRAWN_AT DATETIME NULL, UPDATED_AT DATETIME) ENGINE=InnoDB;
		CREATE TABLE AUTH_PHONE_CLAIM (CANONICAL_PHONE VARCHAR(32) PRIMARY KEY, ACCOUNT_ID INT) ENGINE=InnoDB;
		CREATE TABLE WEO_MEMBER_SOCIAL (USR_SEQ INT, NMS_GATE VARCHAR(10), NMS_STATUS VARCHAR(20)) ENGINE=InnoDB;
		CREATE TABLE ALUMNI_SOCIAL_CREDENTIAL (USR_SEQ INT, PROVIDER VARCHAR(10), ENCRYPTED_CREDENTIAL TEXT) ENGINE=InnoDB;
		CREATE TABLE ALUMNI_SOCIAL_REVOCATION_OUTBOX (
			OUTBOX_ID BIGINT AUTO_INCREMENT PRIMARY KEY, USR_SEQ INT, PROVIDER VARCHAR(10), ACTION VARCHAR(30),
			STATUS VARCHAR(20), ATTEMPT_COUNT INT, NEXT_ATTEMPT_AT DATETIME, LAST_ERROR VARCHAR(500) NULL,
			CREATED_AT DATETIME, UPDATED_AT DATETIME
		) ENGINE=InnoDB;

		INSERT INTO WEO_MEMBER VALUES (42,'member','홍길동','(010) 1234-5678','member@example.com','hash','nick',NULL,'18','영어',NULL,'회사','소개','주소','교사','Y','Y','CCC',NULL,NULL);
		INSERT INTO WEO_ORDER VALUES (1,42,42,NULL,NULL,NULL,NULL,NULL,NULL,'2026-08-01','2026-08-01 00:00:00');
		INSERT INTO ALUMNI_MESSAGE VALUES
			(1,0,7,'shared-client-id',NULL,7,'Y','N'),
			(2,42,7,'shared-client-id',42,7,'N','N'),
			(3,7,42,'recipient-message',7,42,'N','N');
		INSERT INTO AUTH_SESSION_FAMILY VALUES (42,'ACTIVE',NULL,NULL,NOW());
		INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN VALUES (42,NULL);
		INSERT INTO WEO_MEMBER_LOG VALUES (42);
		INSERT INTO ALUMNI_PUSH_DEVICE VALUES (42);
		INSERT INTO ALUMNI_PUSH_PREFERENCE VALUES (42);
		INSERT INTO ALUMNI_MEMBER_BLOCK VALUES (42,7),(7,42);
		INSERT INTO ALUMNI_USER_TAG VALUES (42);
		INSERT INTO ALUMNI_ADMIN_ROLE VALUES (42);
		INSERT INTO ALUMNI_VERIFICATION VALUES (42);
		INSERT INTO AUTH_IDENTITY VALUES (101,42,'member',NULL,'ACTIVE',NULL,NOW());
		INSERT INTO AUTH_PASSWORD_CREDENTIAL VALUES (101);
		INSERT INTO AUTH_ACCOUNT_STATE VALUES (42,'ACTIVE',NULL,NOW());
		INSERT INTO AUTH_PHONE_CLAIM VALUES ('01012345678',42);
		INSERT INTO WEO_MEMBER_SOCIAL VALUES (42,'KT','ACTIVE');
		INSERT INTO ALUMNI_SOCIAL_CREDENTIAL VALUES (42,'KT','encrypted');
	`)
	if err != nil {
		t.Fatal(err)
	}
}
