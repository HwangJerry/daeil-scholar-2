package repository

import (
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"os"
	"strings"
	"testing"
)

func TestProxyIntakeOnMariaDB(t *testing.T) {
	if os.Getenv("AUTOMATIC_ERASURE_DOCKER_INTEGRATION") != "1" {
		t.Skip("isolated database opt-in")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE WEO_MEMBER(USR_SEQ INT PRIMARY KEY,USR_STATUS CHAR(3)) ENGINE=InnoDB;
 CREATE TABLE WEO_MEMBER_SOCIAL(USR_SEQ INT,NMS_GATE CHAR(2)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_ADMIN_ROLE(USR_SEQ INT) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_SOCIAL_REVOCATION_OUTBOX(USR_SEQ INT,PROVIDER VARCHAR(10),ACTION VARCHAR(30),STATUS VARCHAR(20),ATTEMPT_COUNT INT NOT NULL DEFAULT 0,NEXT_ATTEMPT_AT DATETIME,LAST_ERROR VARCHAR(500) NULL,CREATED_AT DATETIME,UPDATED_AT DATETIME) ENGINE=InnoDB;
 INSERT INTO WEO_MEMBER VALUES(5,'CCC'),(9,'CCC');`)
	apply := func(name string) {
		data, e := os.ReadFile("../../migrations/" + name)
		if e != nil {
			t.Fatal(e)
		}
		db.MustExec(string(data))
	}
	for _, name := range []string{"056_create_account_deletion_requests.sql", "057_create_automatic_account_erasure.sql", "059_create_erasure_context.sql", "060_create_erasure_targets.sql", "061_create_erasure_receipt_work.sql", "066_schedule_account_erasure.sql", "067_cancel_account_deletion.sql"} {
		apply(name)
	}
	repo := &AccountDeletionRequestRepository{DB: db, WaitHours: 168}
	receiptKey, cancelKey := strings.Repeat("a", 64), strings.Repeat("c", 64)
	var status string
	var requests int

	// Without the audit table the whole intake rolls back: no request, account untouched.
	if _, e := repo.CreateOnBehalf(5, 9, receiptKey, cancelKey, "email check"); e == nil {
		t.Fatal("intake committed without its audit row")
	}
	db.Get(&status, `SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=5`)
	db.Get(&requests, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_REQUEST`)
	if status != "CCC" || requests != 0 {
		t.Fatal("partial intake left behind", status, requests)
	}
	apply("069_create_account_deletion_intake.sql")

	var invalid *model.ValidationError
	if _, e := repo.CreateOnBehalf(9, 9, receiptKey, cancelKey, "self"); !errors.As(e, &invalid) {
		t.Fatal("operator registered own deletion", e)
	}
	if _, e := repo.CreateOnBehalf(77, 9, receiptKey, cancelKey, "unknown"); !errors.As(e, &invalid) {
		t.Fatal("unknown member not reported", e)
	}
	receipt, e := repo.CreateOnBehalf(5, 9, receiptKey, cancelKey, "email check")
	if e != nil || !receipt.CanCancel || receipt.Status != "pending" {
		t.Fatal("intake failed", receipt, e)
	}
	var audit struct {
		Operator int    `db:"OPERATOR_SEQ"`
		Evidence string `db:"EVIDENCE_REFERENCE"`
	}
	if e = db.Get(&audit, `SELECT OPERATOR_SEQ,EVIDENCE_REFERENCE FROM ALUMNI_ACCOUNT_DELETION_INTAKE WHERE REQUEST_ID=?`, receipt.ID); e != nil || audit.Operator != 9 || audit.Evidence != "email check" {
		t.Fatal("intake audit missing", audit, e)
	}
	db.Get(&status, `SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=5`)
	if status != "AAA" {
		t.Fatal("access not disabled", status)
	}
	if _, e = repo.CreateOnBehalf(5, 9, strings.Repeat("b", 64), cancelKey, "duplicate"); !errors.Is(e, ErrDeletionReceiptConflict) {
		t.Fatal("second intake not rejected", e)
	}
	var audits int
	db.Get(&audits, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_INTAKE`)
	if audits != 1 {
		t.Fatal("duplicate intake audited", audits)
	}
	// The member cancels with the numbers from the operator's reply, exactly as from the app.
	if cancelled, e := repo.Cancel(receiptKey, cancelKey); e != nil || cancelled.Status != "cancelled" {
		t.Fatal("member cannot cancel proxy intake", e)
	}
	db.Get(&status, `SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=5`)
	if status != "CCC" {
		t.Fatal("status not restored", status)
	}
}
