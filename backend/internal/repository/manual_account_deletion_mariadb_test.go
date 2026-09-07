// Disposable MariaDB integration proves requests cannot masquerade as completed erasure.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"os"
	"strings"
	"testing"
	"time"
)

func TestManualAccountDeletionLifecycleOnMariaDB101(t *testing.T) {
	if os.Getenv("MANUAL_DELETION_DOCKER_INTEGRATION") != "1" {
		t.Skip("set MANUAL_DELETION_DOCKER_INTEGRATION=1")
	}
	db := startPasswordResetMariaDB101(t)
	// These fixtures are intentionally unrelated to production and use fake identifiers only.
	db.MustExec(`CREATE TABLE WEO_MEMBER (USR_SEQ INT PRIMARY KEY, USR_STATUS CHAR(3)) ENGINE=InnoDB;
        CREATE TABLE ALUMNI_ADMIN_ROLE (USR_SEQ INT, ROLE VARCHAR(20)) ENGINE=InnoDB;
        CREATE TABLE WEO_MEMBER_SOCIAL (USR_SEQ INT, NMS_GATE CHAR(2)) ENGINE=InnoDB;
        CREATE TABLE ALUMNI_SOCIAL_REVOCATION_OUTBOX (
            ID BIGINT AUTO_INCREMENT PRIMARY KEY, USR_SEQ INT, PROVIDER CHAR(2), ACTION VARCHAR(30),
            STATUS VARCHAR(30), NEXT_ATTEMPT_AT DATETIME, CREATED_AT DATETIME, UPDATED_AT DATETIME) ENGINE=InnoDB;
        CREATE TABLE ALUMNI_MESSAGE (AM_SEQ INT PRIMARY KEY, AM_SENDER_SEQ INT, AM_RECVR_SEQ INT) ENGINE=InnoDB;
        INSERT INTO WEO_MEMBER VALUES (42,'CCC'), (43,'CCC');
        INSERT INTO ALUMNI_ADMIN_ROLE VALUES (42,'root');
        INSERT INTO WEO_MEMBER_SOCIAL VALUES (42,'AP'),(42,'KT');
        INSERT INTO ALUMNI_MESSAGE VALUES (1,42,43);`)
	for _, path := range []string{"../../migrations/055_create_message_reports.sql", "../../migrations/056_create_account_deletion_requests.sql", "../../migrations/057_create_automatic_account_erasure.sql", "../../migrations/059_create_erasure_context.sql", "../../migrations/060_create_erasure_targets.sql", "../../migrations/061_create_erasure_receipt_work.sql"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		db.MustExec(string(data))
		db.MustExec(string(data))
	}
	repo := &AccountDeletionRequestRepository{DB: db}
	hash := strings.Repeat("a", 64)
	receipt, err := repo.Create(42, hash)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "pending" || receipt.TargetAt.Sub(receipt.RequestedAt) != 3*24*time.Hour || receipt.DueAt.Sub(receipt.RequestedAt) != 10*24*time.Hour {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	var status string
	db.Get(&status, "SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=42")
	if status != "AAA" {
		t.Fatal("access still enabled")
	}
	var count int
	db.Get(&count, "SELECT COUNT(*) FROM ALUMNI_ADMIN_ROLE WHERE USR_SEQ=42")
	if count != 0 {
		t.Fatal("admin role retained")
	}
	retry, err := repo.Create(42, hash)
	if err != nil || retry.ID != receipt.ID {
		t.Fatalf("idempotent retry failed: %v", err)
	}
	if _, err = repo.Create(42, strings.Repeat("b", 64)); !errors.Is(err, ErrDeletionReceiptConflict) {
		t.Fatalf("conflict not rejected: %v", err)
	}
	if err = repo.Start(receipt.ID, 42); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("self-review allowed: %v", err)
	}
	if err = repo.Start(receipt.ID, 7); err != nil {
		t.Fatal(err)
	}
	evidence := model.AccountDeletionResolution{EvidenceReference: "disposable-test", RetainedRecords: "없음"}
	if err = repo.Complete(receipt.ID, 7, evidence); !errors.Is(err, ErrDeletionIncomplete) {
		t.Fatalf("status-only deletion completed: %v", err)
	}
	remaining, err := repo.Verify(receipt.ID)
	if err != nil || len(remaining) < 3 {
		t.Fatalf("remaining=%v err=%v", remaining, err)
	}
	// Simulate the operator's actual erasure, NOT just changing a status flag.
	db.MustExec(`DELETE FROM ALUMNI_MESSAGE WHERE AM_SENDER_SEQ=42 OR AM_RECVR_SEQ=42;
        DELETE FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ=42;
        DELETE FROM WEO_MEMBER WHERE USR_SEQ=42;
        UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX SET STATUS='DELIVERED' WHERE USR_SEQ=42 AND PROVIDER='AP';`)
	if err = repo.Complete(receipt.ID, 7, evidence); !errors.Is(err, ErrDeletionIncomplete) {
		t.Fatalf("missing Kakao proof accepted: %v", err)
	}
	db.MustExec("DELETE FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX WHERE USR_SEQ=42 AND PROVIDER='KT'")
	if err = repo.Complete(receipt.ID, 7, evidence); !errors.Is(err, ErrDeletionIncomplete) {
		t.Fatalf("absence of Kakao proof accepted: %v", err)
	}
	db.MustExec(`INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX
        (USR_SEQ,PROVIDER,ACTION,STATUS,NEXT_ATTEMPT_AT,CREATED_AT,UPDATED_AT)
        VALUES (42,'KT','ACCOUNT_DELETE','DELIVERED',UTC_TIMESTAMP(),UTC_TIMESTAMP(),UTC_TIMESTAMP());
        UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX SET CREATED_AT=DATE_SUB(UTC_TIMESTAMP(),INTERVAL 1 DAY) WHERE PROVIDER='AP'`)
	if err = repo.Complete(receipt.ID, 7, evidence); !errors.Is(err, ErrDeletionIncomplete) {
		t.Fatalf("stale Apple proof accepted: %v", err)
	}
	db.MustExec("UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX SET CREATED_AT=UTC_TIMESTAMP() WHERE PROVIDER='AP'")
	if err = repo.Complete(receipt.ID, 7, evidence); err != nil {
		t.Fatal(err)
	}
	final, err := repo.Receipt(hash)
	if err != nil || final.Status != "completed" || final.CompletedAt == nil {
		t.Fatalf("completion: %+v %v", final, err)
	}
	var user sql.NullInt64
	db.Get(&user, "SELECT USR_SEQ FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=?", receipt.ID)
	if user.Valid {
		t.Fatal("receipt retains member reference")
	}
	db.Get(&count, "SELECT COUNT(*) FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX WHERE USR_SEQ=42")
	if count != 0 {
		t.Fatal("provider credentials evidence not scrubbed")
	}
	db.Get(&count, "SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=43")
	if count != 1 {
		t.Fatal("other account changed")
	}
	// Expired receipts and resolved reports are purged; open cases remain.
	db.MustExec(`UPDATE ALUMNI_ACCOUNT_DELETION_REQUEST SET COMPLETED_AT=DATE_SUB(UTC_TIMESTAMP(),INTERVAL 31 DAY);
        INSERT INTO ALUMNI_MESSAGE_REPORT (MESSAGE_ID,REPORTER_SEQ,REPORTED_SEQ,REASON,DETAILS,CONTENT_SNAPSHOT,STATUS,CREATED_AT,RESOLVED_AT)
        VALUES (1,42,43,'spam','','old','removed',DATE_SUB(UTC_TIMESTAMP(),INTERVAL 100 DAY),DATE_SUB(UTC_TIMESTAMP(),INTERVAL 91 DAY)),
        (2,43,42,'spam','','open','open',DATE_SUB(UTC_TIMESTAMP(),INTERVAL 100 DAY),NULL),
        (3,43,42,'spam','','recent','dismissed',UTC_TIMESTAMP(),UTC_TIMESTAMP());`)
	if err = repo.PurgeExpiredPrivacyRecords(context.Background(), 100); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Receipt(hash); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("expired receipt survived")
	}
	db.Get(&count, "SELECT COUNT(*) FROM ALUMNI_MESSAGE_REPORT")
	if count != 2 {
		t.Fatal("retention deleted open or recent report")
	}
	// A failed receipt insert cannot disable another account.
	db.MustExec("DROP TABLE ALUMNI_ERASURE_RECEIPT_WORK")
	db.MustExec("DROP TABLE ALUMNI_ERASURE_TARGET")
	db.MustExec("DROP TABLE ALUMNI_ACCOUNT_DELETION_REQUEST")
	if _, err = repo.Create(43, hash); err == nil {
		t.Fatal("broken storage accepted request")
	}
	db.Get(&status, "SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=43")
	if status != "CCC" {
		t.Fatal("failed request changed account")
	}
	migration, err := os.ReadFile("../../migrations/056_create_account_deletion_requests.sql")
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(string(migration))
	targetMigration, err := os.ReadFile("../../migrations/060_create_erasure_targets.sql")
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(string(targetMigration))
	receiptMigration, err := os.ReadFile("../../migrations/061_create_erasure_receipt_work.sql")
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(string(receiptMigration))
	db.MustExec("DROP TABLE ALUMNI_ADMIN_ROLE")
	if _, err = repo.Create(43, hash); err == nil {
		t.Fatal("failed role removal accepted request")
	}
	db.Get(&status, "SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=43")
	db.Get(&count, "SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_REQUEST")
	if status != "CCC" || count != 0 {
		t.Fatal("failure after access change did not roll back both changes")
	}
}
