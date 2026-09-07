// donation_retention_source_mariadb_test.go — Exercise corrected accounting facts across deletion boundaries.
package repository

import (
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDonationRetentionSourceOnMariaDB101(t *testing.T) {
	if os.Getenv("AUTOMATIC_ERASURE_DOCKER_INTEGRATION") != "1" {
		t.Skip("set AUTOMATIC_ERASURE_DOCKER_INTEGRATION=1")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE WEO_MEMBER (USR_SEQ INT PRIMARY KEY,USR_STATUS CHAR(3),USR_ID VARCHAR(50),USR_NAME VARCHAR(100),USR_EMAIL VARCHAR(100),USR_PHONE VARCHAR(32),USR_PHOTO VARCHAR(200),USR_BIZ_CARD VARCHAR(200)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    CREATE TABLE WEO_MEMBER_SOCIAL (USR_SEQ INT,NMS_GATE CHAR(2),NMS_ID VARCHAR(100)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    CREATE TABLE ALUMNI_ADMIN_ROLE (USR_SEQ INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    CREATE TABLE ALUMNI_SOCIAL_REVOCATION_OUTBOX (ID BIGINT AUTO_INCREMENT PRIMARY KEY,USR_SEQ INT,PROVIDER CHAR(2),ACTION VARCHAR(30),STATUS VARCHAR(30),NEXT_ATTEMPT_AT DATETIME,CREATED_AT DATETIME,UPDATED_AT DATETIME) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    CREATE TABLE ALUMNI_MESSAGE (AM_SEQ INT PRIMARY KEY,AM_SENDER_SEQ INT,AM_RECVR_SEQ INT,AM_CONTENT TEXT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    CREATE TABLE WEO_VISIT_DAILY (VD_USR_SEQ INT,VD_VISITOR_ID VARCHAR(50)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    CREATE TABLE WEO_ORDER (O_SEQ INT PRIMARY KEY,USR_SEQ INT,O_ACCOUNT_USR_SEQ INT,O_DONOR_NAME VARCHAR(100),O_DONOR_PHONE VARCHAR(32),O_DONATION_DATE DATE,O_GROSS_AMOUNT BIGINT,O_REFUNDED_AMOUNT BIGINT,O_NET_RECEIVED_AMOUNT BIGINT,O_SOURCE VARCHAR(30),O_TRANSACTION_NO VARCHAR(100),O_LIFECYCLE_STATUS VARCHAR(30),O_TYPE CHAR(1)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    CREATE TABLE WEO_PG_DATA (O_SEQ INT,NUM_CARD VARCHAR(50)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    INSERT INTO WEO_MEMBER VALUES (42,'CCC','fake42','Synthetic Donor','fake42@example.org','01000000042','/uploads/profile/42.jpg','/upload/legacy-card.jpg'),(43,'CCC','fake43','Other Donor','fake43@example.org','01000000043','','');
    INSERT INTO WEO_MEMBER_SOCIAL VALUES (42,'AP','synthetic-subject');
    INSERT INTO ALUMNI_MESSAGE VALUES (1,42,43,'outbound'),(2,43,42,'inbound'),(3,43,43,'unrelated');
    INSERT INTO WEO_VISIT_DAILY VALUES (42,'visitor42'),(43,'visitor43');
    INSERT INTO WEO_ORDER VALUES (1,42,42,'Synthetic Donor','01000000042','2025-01-01',100,0,100,'happy_nanum','fake-tx-1','completed','A'),(2,43,43,'Other Donor','01000000043','2025-01-01',50,0,50,'happy_nanum','fake-tx-2','completed','A');
    INSERT INTO WEO_PG_DATA VALUES (1,'fake-card'),(2,'other-card');`)
	db.MustExec(`ALTER TABLE WEO_MEMBER ADD USR_THUMNAIL TEXT`)
	for _, name := range []string{"055_create_message_reports.sql", "056_create_account_deletion_requests.sql", "057_create_automatic_account_erasure.sql", "059_create_erasure_context.sql", "060_create_erasure_targets.sql", "061_create_erasure_receipt_work.sql", "062_create_profile_file_history.sql", "063_bind_donation_retention_source.sql"} {
		data, err := os.ReadFile("../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		db.MustExec(string(data))
	}

	db.MustExec(`UPDATE WEO_MEMBER SET USR_PHOTO='',USR_BIZ_CARD='',USR_THUMNAIL=''; DELETE FROM WEO_MEMBER_SOCIAL;
 UPDATE WEO_ORDER SET O_DONATION_DATE='2010-01-01' WHERE O_SEQ=1;
 ALTER TABLE WEO_ORDER ADD O_ACCOUNT_UNLINKED_AT DATETIME NULL, ADD O_COMPOSITE_KEY VARCHAR(200), ADD O_DONOR_COHORT VARCHAR(30), ADD O_DONOR_DEPARTMENT VARCHAR(30), ADD O_GATE VARCHAR(30), ADD O_PAYMENT_METHOD VARCHAR(30), ADD O_MEMO TEXT, ADD O_PRICE BIGINT, ADD O_PAY BIGINT, ADD O_PAY_TYPE VARCHAR(10), ADD O_STATUS VARCHAR(10), ADD O_PAYMENT VARCHAR(10), ADD EDT_OPER INT, ADD EDT_DATE DATETIME, ADD EDT_IPADDR VARCHAR(100), ADD REG_DATE DATETIME;
 UPDATE WEO_ORDER SET REG_DATE='2026-09-01 00:00:00';`)
	repo := &AccountDeletionRequestRepository{DB: db}
	receipt, err := repo.Create(42, strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.Start(receipt.ID, 0); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2010, 12, 31, 0, 0, 0, 0, time.UTC)
	until := start.AddDate(10, 0, 0)
	originalSource, err := repo.DonationRetentionSource(1)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveDonationRetention(model.DonationRetentionDecision{SourceFingerprint: originalSource, OrderID: 1, Basis: "ledger_10y", BasisDate: &start, Until: &until, Evidence: "synthetic-reviewed-2010"}); err != nil {
		t.Fatal(err)
	}
	if err = repo.ResolveReceiptWork(receipt.ID, 7, model.AccountDeletionResolution{ReceiptWorkStatus: "active", OriginalStorage: "separate_excel", EvidenceReference: "synthetic-contact-secured"}); err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveErasureContext(receipt.ID, []byte("synthetic-ciphertext")); err != nil {
		t.Fatal(err)
	}
	var status string
	db.Get(&status, `SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=42`)
	if status != "AAA" {
		t.Fatal(status)
	}
	donationRepo := &AdminDonationRepository{DB: db}
	order := model.NormalizedDonationOrder{DonationOrderInput: model.DonationOrderInput{Source: "happy_nanum", DonationDate: "2025-01-01", Donor: model.DonationDonor{Name: "Synthetic Donor", Phone: "01000000042"}, GrossAmount: 100, Status: "completed", PaymentMethod: "bank_transfer", LastEditedAt: "2026-09-01T00:00:00Z"}, NetReceivedAmount: 100, LegacyGate: "H", LegacyStatus: "Y", LegacyPayment: "Y"}
	if err = donationRepo.UpdateDonationOrder(1, order, 7, "127.0.0.1"); err != nil {
		t.Fatalf("actual AAA update failed: %v", err)
	}

	var date string
	var decisions int
	db.Get(&date, `SELECT DATE_FORMAT(O_DONATION_DATE,'%Y-%m-%d') FROM WEO_ORDER WHERE O_SEQ=1`)
	db.Get(&decisions, `SELECT COUNT(*) FROM ALUMNI_DONATION_RETENTION WHERE O_SEQ=1`)
	if date != "2025-01-01" || decisions != 0 {
		t.Fatal("AAA date replacement did not invalidate old review", date, decisions)
	}
	stale := model.DonationRetentionDecision{SourceFingerprint: originalSource, OrderID: 1, Basis: "ledger_10y", BasisDate: &start, Until: &until, Evidence: "stale-review"}
	if err = repo.SaveDonationRetention(stale); err == nil {
		t.Fatal("stale review accepted after edit")
	}
	validate := func(d model.DonationRetentionDecision) error {
		if d.Basis != "ledger_10y" || d.BasisDate == nil || d.Until == nil || !d.Until.Equal(d.BasisDate.AddDate(10, 0, 0)) {
			return errors.New("invalid retention")
		}
		return nil
	}
	w := model.ErasureWork{RequestID: receipt.ID, UserSeq: 42, Stage: "queued"}
	if err = repo.PrepareErasure(w, validate); err == nil {
		t.Fatal("invalidated review passed prepare")
	}
	if err = repo.EraseDatabase(w, nil, validate); err == nil {
		t.Fatal("invalidated review passed final deletion")
	}
	source, err := repo.DonationRetentionSource(1)
	if err != nil {
		t.Fatal(err)
	}
	start = time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	until = start.AddDate(10, 0, 0)
	current := model.DonationRetentionDecision{SourceFingerprint: source, OrderID: 1, Basis: "ledger_10y", BasisDate: &start, Until: &until, Evidence: "new-review"}
	if err = repo.SaveDonationRetention(current); err != nil {
		t.Fatal(err)
	}
	if err = repo.PrepareErasure(w, validate); err != nil {
		t.Fatal(err)
	}
	subject, err := repo.ErasureExternalSubject(w)
	if err != nil {
		t.Fatal(err)
	}
	w.ContextRetentions = subject.Retentions
	// A reassignment removes the row from this user's current query, but its old
	// external handoff must not become permission to process another donor's record.
	db.MustExec(`UPDATE WEO_ORDER SET USR_SEQ=43,O_ACCOUNT_USR_SEQ=43 WHERE O_SEQ=1`)
	if err = repo.EraseDatabase(w, nil, validate); err == nil {
		t.Fatal("extra retained handoff survived donation reassignment")
	}
	db.MustExec(`UPDATE WEO_ORDER SET USR_SEQ=42,O_ACCOUNT_USR_SEQ=42 WHERE O_SEQ=1`)
	// Model a concurrent correction after context capture, followed by fresh review.
	// This also catches legacy SQL/payment writers that do not run the admin invalidation.
	db.MustExec(`UPDATE WEO_ORDER SET O_DONOR_NAME='Corrected Donor' WHERE O_SEQ=1`)
	if err = repo.EraseDatabase(w, nil, validate); err == nil {
		t.Fatal("source changed between prepare and deletion")
	}
	current.SourceFingerprint, err = repo.DonationRetentionSource(1)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveDonationRetention(current); err != nil {
		t.Fatal(err)
	}
	if err = repo.EraseDatabase(w, nil, validate); err == nil {
		t.Fatal("new review accepted with stale external handoff")
	}
	var expires string
	db.Get(&expires, `SELECT DATE_FORMAT(EXPIRES_AT,'%Y-%m-%d %H:%i:%s') FROM ALUMNI_ERASURE_CONTEXT WHERE REQUEST_ID=?`, receipt.ID)
	if err = repo.RefreshErasureContext(receipt.ID, []byte("refreshed-ciphertext")); err != nil {
		t.Fatal(err)
	}
	var refreshedExpires string
	db.Get(&refreshedExpires, `SELECT DATE_FORMAT(EXPIRES_AT,'%Y-%m-%d %H:%i:%s') FROM ALUMNI_ERASURE_CONTEXT WHERE REQUEST_ID=?`, receipt.ID)
	if expires != refreshedExpires {
		t.Fatal("context refresh extended identifier lifetime")
	}
	subject, err = repo.ErasureExternalSubject(w)
	if err != nil {
		t.Fatal(err)
	}
	w.ContextRetentions = subject.Retentions
	sealed := 0
	if err = repo.EraseDatabase(w, func(data []byte) ([]byte, error) { sealed++; return data, nil }, validate); err != nil {
		t.Fatal(err)
	}
	var orders, archives, members int
	db.Get(&orders, `SELECT COUNT(*) FROM WEO_ORDER WHERE O_SEQ=1`)
	db.Get(&archives, `SELECT COUNT(*) FROM ALUMNI_DONATION_LEGAL_ARCHIVE`)
	db.Get(&members, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=42`)
	if orders != 0 || archives != 1 || sealed != 1 || members != 0 {
		t.Fatal("current review did not archive exactly once", orders, archives, sealed, members)
	}
	if err = repo.SaveDonationRetention(current); err == nil {
		t.Fatal("review resurrected after donation deletion")
	}
	if err = repo.RefreshErasureContext(receipt.ID, []byte("after-database-delete")); err == nil {
		t.Fatal("context changed after irreversible erasure")
	}
	// A review writer waiting behind an accounting edit must validate the new row,
	// not save the decision against the old snapshot after the editor commits.
	oldSource, err := repo.DonationRetentionSource(2)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE WEO_ORDER SET O_DONOR_NAME='Concurrent edit' WHERE O_SEQ=2`); err != nil {
		t.Fatal(err)
	}
	pending := make(chan error, 1)
	go func() {
		pending <- repo.SaveDonationRetention(model.DonationRetentionDecision{OrderID: 2, SourceFingerprint: oldSource, Basis: "erase", Evidence: "outdated-review"})
	}()
	select {
	case err := <-pending:
		t.Fatalf("review bypassed order lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-pending:
		if err == nil {
			t.Fatal("concurrent stale review committed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("review did not finish after edit commit")
	}

}
