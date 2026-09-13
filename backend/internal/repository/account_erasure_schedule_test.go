package repository

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestScheduledErasureOnMariaDB(t *testing.T) {
	if os.Getenv("AUTOMATIC_ERASURE_DOCKER_INTEGRATION") != "1" {
		t.Skip("isolated database opt-in required")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE WEO_MEMBER(USR_SEQ INT PRIMARY KEY,USR_STATUS CHAR(3)) ENGINE=InnoDB;
 CREATE TABLE WEO_MEMBER_SOCIAL(USR_SEQ INT,NMS_GATE CHAR(2)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_ADMIN_ROLE(USR_SEQ INT) ENGINE=InnoDB;
 INSERT INTO WEO_MEMBER VALUES(42,'CCC'),(43,'CCC');`)
	for _, name := range []string{"056_create_account_deletion_requests.sql", "057_create_automatic_account_erasure.sql", "059_create_erasure_context.sql", "060_create_erasure_targets.sql", "061_create_erasure_receipt_work.sql", "066_schedule_account_erasure.sql"} {
		data, err := os.ReadFile("../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		db.MustExec(string(data))
	}
	repo := &AccountDeletionRequestRepository{DB: db, WaitHours: 72}
	receipt, err := repo.Create(42, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ScheduledAt == nil || receipt.ScheduledAt.Sub(receipt.RequestedAt) != 72*time.Hour {
		t.Fatal("schedule not captured", receipt)
	}
	assertBatch := func(n int) {
		t.Helper()
		batch, e := repo.ErasureBatch(context.Background())
		if e != nil || len(batch) != n {
			t.Fatalf("batch want %d got %v %v", n, batch, e)
		}
	}
	assertBatch(0)
	// A mode switch / retry cannot bypass a future schedule.
	if err = repo.SetErasureMode(receipt.ID, 7, "automatic"); err != nil {
		t.Fatal(err)
	}
	assertBatch(0)
	// Config changes and duplicate request retries must not move the accepted schedule.
	repo.WaitHours = 24
	again, err := repo.Create(42, strings.Repeat("a", 64))
	if err != nil || !again.ScheduledAt.Equal(*receipt.ScheduledAt) {
		t.Fatal("schedule moved", err)
	}
	if err = repo.ControlSchedule(receipt.ID, 42, "expedite"); err == nil {
		t.Fatal("self approval accepted")
	}
	repo.TestUserSeq = 43
	if err = repo.ControlSchedule(receipt.ID, 7, "expedite"); err == nil {
		t.Fatal("scope bypass")
	}
	repo.TestUserSeq = 0
	// Boundary is inclusive and independent of the retry timestamp.
	db.MustExec(`UPDATE ALUMNI_ERASURE_SCHEDULE SET SCHEDULED_AT=DATE_ADD(UTC_TIMESTAMP(),INTERVAL 1 HOUR)`)
	assertBatch(0)
	db.MustExec(`UPDATE ALUMNI_ERASURE_SCHEDULE SET SCHEDULED_AT=UTC_TIMESTAMP()`)
	assertBatch(1)
	db.MustExec(`UPDATE ALUMNI_ERASURE_SCHEDULE SET SCHEDULED_AT=DATE_ADD(UTC_TIMESTAMP(),INTERVAL 72 HOUR)`)
	assertBatch(0)
	if err = repo.ControlSchedule(receipt.ID, 7, "expedite"); err != nil {
		t.Fatal(err)
	}
	assertBatch(1)
	// Duplicate clicks preserve the original operator and audit timestamp.
	if err = repo.ControlSchedule(receipt.ID, 8, "expedite"); err != nil {
		t.Fatal(err)
	}
	var operator int
	db.Get(&operator, `SELECT EXPEDITED_BY FROM ALUMNI_ERASURE_SCHEDULE WHERE REQUEST_ID=?`, receipt.ID)
	if operator != 7 {
		t.Fatal("audit overwritten")
	}
	release, locked, err := repo.ErasureLock(context.Background())
	if err != nil || !locked {
		t.Fatal(err)
	}
	if err = repo.ControlSchedule(receipt.ID, 7, "expedite"); err == nil {
		t.Fatal("concurrent operator accepted")
	}
	release()
	// Unscheduled historical requests remain untouched until explicit review.
	db.MustExec(`DELETE FROM ALUMNI_ERASURE_SCHEDULE WHERE REQUEST_ID=?`, receipt.ID)
	assertBatch(0)
	if err = repo.ControlSchedule(receipt.ID, 7, "schedule"); err != nil {
		t.Fatal(err)
	}
	assertBatch(0)
}
