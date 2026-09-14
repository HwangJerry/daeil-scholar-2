// social_revocation_erasure_mariadb_test.go — Erasure provider revocation drains through the real outbox schema.
package repository

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

func TestErasureSocialRevocationClaimOnMariaDB101(t *testing.T) {
	if os.Getenv("AUTOMATIC_ERASURE_DOCKER_INTEGRATION") != "1" {
		t.Skip("set AUTOMATIC_ERASURE_DOCKER_INTEGRATION=1")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE WEO_MEMBER (USR_SEQ INT PRIMARY KEY,USR_STATUS CHAR(3),USR_ID VARCHAR(50),USR_NAME VARCHAR(100),USR_EMAIL VARCHAR(100),USR_PHONE VARCHAR(32),USR_PHOTO VARCHAR(200),USR_BIZ_CARD VARCHAR(200),USR_THUMNAIL TEXT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE WEO_MEMBER_SOCIAL (USR_SEQ INT,NMS_GATE CHAR(2),NMS_ID VARCHAR(100)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE ALUMNI_ADMIN_ROLE (USR_SEQ INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
INSERT INTO WEO_MEMBER VALUES (42,'CCC','fake42','Synthetic Member','fake42@example.org','01000000042','','',NULL),(43,'CCC','fake43','Other Member','fake43@example.org','01000000043','','',NULL);
INSERT INTO WEO_MEMBER_SOCIAL VALUES (42,'KT','synthetic-kakao');`)
	// The production outbox schema, including its nullable LAST_ERROR column.
	social, err := os.ReadFile("../../migrations/045_reconcile_social_credential_and_revocation_outbox.sql")
	if err != nil {
		t.Fatal(err)
	}
	ddl := strings.NewReplacer("DELIMITER //", "", "DELIMITER ;", "", "END //", "END;").Replace(string(social))
	if _, err = db.Exec(ddl); err != nil {
		t.Fatalf("045: %v", err)
	}
	for _, name := range []string{"055_create_message_reports.sql", "056_create_account_deletion_requests.sql", "057_create_automatic_account_erasure.sql", "066_schedule_account_erasure.sql", "067_cancel_account_deletion.sql", "059_create_erasure_context.sql", "060_create_erasure_targets.sql", "061_create_erasure_receipt_work.sql", "062_create_profile_file_history.sql", "063_bind_donation_retention_source.sql"} {
		data, err := os.ReadFile("../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(data)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// A row written before error messages were stored explicitly.
	db.MustExec(`INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX (USR_SEQ,PROVIDER,ACTION,STATUS,NEXT_ATTEMPT_AT,LAST_ERROR,CREATED_AT,UPDATED_AT) VALUES (43,'AP','DISCONNECT','PENDING',NOW(),NULL,NOW(),NOW())`)

	repo := &AccountDeletionRequestRepository{WaitHours: 72, DB: db}
	receipt, err := repo.Create(42, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.Start(receipt.ID, 7); err != nil {
		t.Fatal(err)
	}

	// The worker must claim every due row, including ones without an error message.
	auth := NewAuthRepository(db)
	entries, err := auth.ClaimDueSocialRevocations("worker-a", 5*time.Minute, 20)
	if err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("claimed %d rows, want 2", len(entries))
	}
	var outboxID int64
	for _, entry := range entries {
		if entry.LastError != "" {
			t.Fatalf("unexpected error message %q", entry.LastError)
		}
		if entry.Action == "ACCOUNT_DELETE" && entry.USRSeq == 42 {
			outboxID = entry.OutboxID
		}
	}
	if outboxID == 0 {
		t.Fatal("erasure revocation row was not claimed")
	}
	var unset int
	if err = db.Get(&unset, `SELECT COUNT(*) FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX WHERE USR_SEQ=42 AND ACTION='ACCOUNT_DELETE' AND (LAST_ERROR IS NULL OR ATTEMPT_COUNT<>0)`); err != nil || unset != 0 {
		t.Fatalf("erasure revocation row stored without explicit defaults: %d %v", unset, err)
	}

	// An operator retry releases a claim left behind by a failed worker run.
	if err = repo.ControlSchedule(receipt.ID, 7, "retry_social"); err != nil {
		t.Fatalf("retry of a stuck claim failed: %v", err)
	}
	var released int
	if err = db.Get(&released, `SELECT COUNT(*) FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX WHERE OUTBOX_ID=? AND STATUS='PENDING' AND CLAIM_TOKEN IS NULL`, outboxID); err != nil || released != 1 {
		t.Fatalf("stuck claim not released: %d %v", released, err)
	}

	// Completing the provider call must satisfy the erasure's provider proof.
	if err = auth.MarkSocialRevocationRevoked(outboxID); err != nil {
		t.Fatal(err)
	}
	if err = auth.CompleteAccountDeletionRevocation(42, "KT"); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = verifyDeletionProviders(tx, receipt.ID, 42); err != nil {
		t.Fatalf("provider proof not recognized: %v", err)
	}
	// Once delivered there is nothing to retry, and the operator is told so.
	var invalid *model.ValidationError
	if err = repo.ControlSchedule(receipt.ID, 7, "retry_social"); !errors.As(err, &invalid) {
		t.Fatalf("retry with nothing pending = %v, want a validation message", err)
	}
}
