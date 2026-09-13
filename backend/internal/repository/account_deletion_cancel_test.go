package repository

import (
	"context"
	"github.com/dflh-saf/backend/internal/model"
	"os"
	"strings"
	"testing"
)

func TestCancellationOnMariaDB(t *testing.T) {
	if os.Getenv("AUTOMATIC_ERASURE_DOCKER_INTEGRATION") != "1" {
		t.Skip("isolated database opt-in")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE WEO_MEMBER(USR_SEQ INT PRIMARY KEY,USR_STATUS CHAR(3)) ENGINE=InnoDB;
 CREATE TABLE WEO_MEMBER_SOCIAL(USR_SEQ INT,NMS_GATE CHAR(2)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_ADMIN_ROLE(USR_SEQ INT) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_SOCIAL_REVOCATION_OUTBOX(USR_SEQ INT,PROVIDER VARCHAR(10),ACTION VARCHAR(30),STATUS VARCHAR(20),NEXT_ATTEMPT_AT DATETIME,CREATED_AT DATETIME,UPDATED_AT DATETIME) ENGINE=InnoDB;
 INSERT INTO WEO_MEMBER VALUES(1,'CCC'),(2,'BBB'),(3,'BAA'),(4,'ZZZ'),(5,'CCC'),(6,'CCC');
 INSERT INTO ALUMNI_ADMIN_ROLE VALUES(4);`)
	for _, name := range []string{"056_create_account_deletion_requests.sql", "057_create_automatic_account_erasure.sql", "059_create_erasure_context.sql", "060_create_erasure_targets.sql", "061_create_erasure_receipt_work.sql", "066_schedule_account_erasure.sql", "067_cancel_account_deletion.sql"} {
		data, e := os.ReadFile("../../migrations/" + name)
		if e != nil {
			t.Fatal(e)
		}
		db.MustExec(string(data))
	}
	repo := &AccountDeletionRequestRepository{DB: db, WaitHours: 72}
	for i, want := range map[int]string{1: "CCC", 2: "BBB", 3: "BAA", 4: "CCC"} {
		receiptKey := strings.Repeat(string(rune('a'+i)), 64)
		secret := strings.Repeat("f", 64)
		receipt, e := repo.CreateCancelable(i, receiptKey, secret)
		if e != nil {
			t.Fatal(e)
		}
		if !receipt.CanCancel {
			t.Fatal("new request not cancellable")
		}
		// Retry must not trip the disabled-account restoration guard.
		if _, e = repo.CreateCancelable(i, receiptKey, secret); e != nil {
			t.Fatal(e)
		}
		if _, e = repo.Cancel(receiptKey, strings.Repeat("0", 64)); e == nil {
			t.Fatal("lookup authority cancelled request")
		}
		if e = repo.ControlSchedule(receipt.ID, 99, "expedite"); e != nil {
			t.Fatal(e)
		}
		result, e := repo.Cancel(receiptKey, secret)
		if e != nil || result.Status != "cancelled" || result.CanCancel || result.CancelledAt == nil {
			t.Fatal("cancel failed", result, e)
		}
		var status string
		db.Get(&status, `SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=?`, i)
		if status != want {
			t.Fatal("wrong status restored", i, status)
		}
		if _, e = repo.Cancel(receiptKey, secret); e != nil {
			t.Fatal("duplicate cancellation failed", e)
		}
		if e = repo.ControlSchedule(receipt.ID, 99, "expedite"); e == nil {
			t.Fatal("cancelled request reactivated")
		}
		if e = repo.Start(receipt.ID, 0); e == nil {
			t.Fatal("cancelled request started")
		}
		if active, e := repo.ErasureActive(receipt.ID); e != nil || active {
			t.Fatal("cancelled work active")
		}
		if e = repo.ReviewErasureTarget(receipt.ID, 99, model.ErasureTarget{Name: "backups", Status: "complete", Evidence: "fake"}); e == nil {
			t.Fatal("cancelled proof changed")
		}
		next, e := repo.CreateCancelable(i, strings.Repeat(string(rune('0'+i)), 64), secret)
		if e != nil || next.ID == receipt.ID {
			t.Fatal("reapplication failed", e)
		}
	}
	var roles int
	db.Get(&roles, `SELECT COUNT(*) FROM ALUMNI_ADMIN_ROLE WHERE USR_SEQ=4`)
	if roles != 0 {
		t.Fatal("admin role resurrected")
	}
	// A subsequent moderation action forbids automatic restoration, even AAA -> AAA.
	blocked, e := repo.CreateCancelable(5, strings.Repeat("9", 64), strings.Repeat("f", 64))
	if e != nil {
		t.Fatal(e)
	}
	db.MustExec(`UPDATE WEO_MEMBER SET USR_STATUS='AAA' WHERE USR_SEQ=5`)
	if _, e = repo.Cancel(strings.Repeat("9", 64), strings.Repeat("f", 64)); e == nil {
		t.Fatal("moderation overridden")
	}
	_ = blocked
	started, e := repo.CreateCancelable(6, strings.Repeat("8", 64), strings.Repeat("f", 64))
	if e != nil {
		t.Fatal(e)
	}
	if e = repo.Start(started.ID, 99); e != nil {
		t.Fatal(e)
	}
	if _, e = repo.Cancel(strings.Repeat("8", 64), strings.Repeat("f", 64)); e == nil {
		t.Fatal("started deletion cancelled")
	}
	release, locked, e := repo.ErasureLock(context.Background())
	if e != nil || !locked {
		t.Fatal(e)
	}
	if _, e = repo.Cancel(strings.Repeat("1", 64), strings.Repeat("f", 64)); e == nil {
		t.Fatal("worker lock bypassed")
	}
	release()
}
