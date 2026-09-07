// automatic_erasure_mariadb_test.go — Real transactional erasure of synthetic accounts.
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

func TestAutomaticErasureOnMariaDB101(t *testing.T) {
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

	// Existing paths without WEO_FILES rows survive replacements until erasure.
	profileRepo := &ProfileRepository{DB: db}
	if err := profileRepo.AssignProfileUpload(42, 101, "/uploads/profile/new42.jpg", false); err != nil {
		t.Fatal(err)
	}
	var historyCount int
	if err := db.Get(&historyCount, `SELECT COUNT(*) FROM ALUMNI_PROFILE_FILE_HISTORY WHERE USR_SEQ=42`); err != nil || historyCount != 2 {
		t.Fatal("prior profile reference lost", err)
	}
	// Exercise the actual legacy engine transition twice without touching unrelated tables.
	db.MustExec(`ALTER TABLE WEO_ORDER ENGINE=MyISAM; CREATE TABLE UNRELATED_LEGACY (ID INT) ENGINE=MyISAM; INSERT INTO UNRELATED_LEGACY VALUES (1)`)
	ddl, err := os.ReadFile("../../migrations/058_convert_erasure_tables_to_innodb.sql")
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(string(ddl), "DELIMITER //", ""), "DELIMITER ;", ""), "END//", "END;")
	db.MustExec(migrationSQL)
	db.MustExec(migrationSQL)
	var engine string
	if err = db.Get(&engine, `SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_ORDER'`); err != nil || engine != "InnoDB" {
		t.Fatal("conversion failed", engine, err)
	}
	if err = db.Get(&engine, `SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='UNRELATED_LEGACY'`); err != nil || engine != "MyISAM" {
		t.Fatal("unrelated table converted")
	}
	repo := &AccountDeletionRequestRepository{DB: db}
	receipt, err := repo.Create(42, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	work := model.ErasureWork{RequestID: receipt.ID, UserSeq: 42, Stage: "queued"}
	if err = profileRepo.UpdateProfilePhoto(42, "/uploads/forbidden-after-withdrawal.jpg"); err == nil {
		t.Fatal("withdrawn member changed profile history")
	}
	if err = repo.SetErasureMode(receipt.ID, 42, "manual"); err == nil {
		t.Fatal("self operation allowed")
	}
	if err = repo.SetErasureMode(receipt.ID, 7, "manual"); err != nil {
		t.Fatal(err)
	}
	batch, err := repo.ErasureBatch(context.Background())
	if err != nil || len(batch) != 0 {
		t.Fatal("manual request selected for automation")
	}
	if err = repo.SetErasureMode(receipt.ID, 7, "automatic"); err != nil {
		t.Fatal(err)
	}
	batch, err = repo.ErasureBatch(context.Background())
	if err != nil || len(batch) != 1 {
		t.Fatal("automatic resume not scheduled")
	}

	release, locked, err := repo.ErasureLock(context.Background())
	if err != nil || !locked {
		t.Fatal("lock", err)
	}
	_, second, err := repo.ErasureLock(context.Background())
	if err != nil || second {
		t.Fatal("two workers acquired lock")
	}
	release()
	if err = repo.Start(receipt.ID, 0); err != nil {
		t.Fatal(err)
	}
	validate := func(d model.DonationRetentionDecision) error {
		if d.Basis != "ledger_10y" {
			return errors.New("invalid")
		}
		return nil
	}
	if err = repo.PrepareErasure(work, validate); err == nil {
		t.Fatal("revocation ignored")
	}
	db.MustExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX SET STATUS='DELIVERED'`)
	if err = repo.PrepareErasure(work, validate); err == nil {
		t.Fatal("missing legal decision ignored")
	}
	templateCalled := false
	repo.DonationRetentionTemplate = func(id int, date time.Time) (model.DonationRetentionDecision, error) {
		templateCalled = true
		return model.DonationRetentionDecision{}, nil
	}
	if err = repo.PrepareErasure(work, validate); err == nil || templateCalled {
		t.Fatal("registration date used as statutory retention clock")
	}
	repo.DonationRetentionTemplate = nil
	start := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	until := start.AddDate(10, 0, 0)
	if err = repo.ResolveReceiptWork(receipt.ID, 7, model.AccountDeletionResolution{ReceiptWorkStatus: "active", OriginalStorage: "separate_excel", ContactSecured: true, EvidenceReference: "synthetic-pending-receipt"}); err != nil {
		t.Fatal(err)
	}
	fingerprint, err := repo.DonationRetentionSource(1)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveDonationRetention(model.DonationRetentionDecision{SourceFingerprint: fingerprint, OrderID: 1, Basis: "ledger_10y", BasisDate: &start, Until: &until, Evidence: "synthetic-review"}); err != nil {
		t.Fatal(err)
	}
	if err = repo.PrepareErasure(work, validate); err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveErasureContext(receipt.ID, []byte("synthetic-encrypted-context")); err != nil {
		t.Fatal(err)
	}
	subject, err := repo.ErasureExternalSubject(work)
	if err != nil {
		t.Fatal(err)
	}
	work.ContextRetentions = subject.Retentions
	targets, err := repo.ErasureTargets(receipt.ID)
	if err != nil || len(targets) != 4 {
		t.Fatal("targets not initialized", err)
	}
	if err = repo.RecordErasureTargets(receipt.ID, []model.ErasureTarget{{Name: "external_data", Status: "not_applicable", Evidence: "synthetic-inventory"}}); err != nil {
		t.Fatal(err)
	}
	if err = repo.SetErasureMode(receipt.ID, 7, "manual"); err != nil {
		t.Fatal(err)
	}
	if err = repo.BeginErasureTargets(receipt.ID); err == nil {
		t.Fatal("manual target processed")
	}
	if err = repo.RecordErasureTargets(receipt.ID, []model.ErasureTarget{{Name: "backups", Status: "complete", Evidence: "unapproved"}}); err == nil {
		t.Fatal("manual takeover overwritten")
	}
	if err = repo.SetErasureMode(receipt.ID, 7, "automatic"); err != nil {
		t.Fatal(err)
	}
	if err = repo.BeginErasureTargets(receipt.ID); err != nil {
		t.Fatal(err)
	}
	if err = repo.RecordErasureTargets(receipt.ID, []model.ErasureTarget{{Name: "external_data", Status: "failed"}, {Name: "backups", Status: "pending"}}); err != nil {
		t.Fatal(err)
	}
	targets, err = repo.ErasureTargets(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if target.Name == "external_data" && (target.Status != "not_applicable" || target.Attempts != 0) {
			t.Fatal("verified target regressed", target)
		}
	}
	// Rollout allowlist excludes other queued accounts without changing their mode.
	repo.TestUserSeq = 43
	if selected, err := repo.ErasureBatch(context.Background()); err != nil || len(selected) != 0 {
		t.Fatal("test rollout selected another account", err)
	}
	repo.TestUserSeq = 42
	if selected, err := repo.ErasureBatch(context.Background()); err != nil || len(selected) != 1 {
		t.Fatal("test rollout skipped approved account", err)
	}
	repo.TestUserSeq = 0
	// Unknown references block and roll back the archive, aggregate and member deletion together.
	db.MustExec(`CREATE TABLE UNHANDLED_REFERENCE (USR_SEQ INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4; INSERT INTO UNHANDLED_REFERENCE VALUES (42)`)
	seal := func(data []byte) ([]byte, error) {
		if strings.Contains(string(data), "phone") || strings.Contains(string(data), "userSeq") || strings.Contains(string(data), "01000000042") {
			t.Fatal("archive copied unnecessary profile data")
		}
		return []byte("synthetic-encrypted-test-output"), nil
	}
	if err = repo.EraseDatabase(work, seal, validate); err == nil {
		t.Fatal("unhandled reference accepted")
	}
	var count int
	db.Get(&count, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=42`)
	if count != 1 {
		t.Fatal("failed erasure did not roll back")
	}
	db.Get(&count, `SELECT COUNT(*) FROM ALUMNI_DONATION_LEGAL_ARCHIVE`)
	if count != 0 {
		t.Fatal("archive did not roll back")
	}
	db.MustExec(`DROP TABLE UNHANDLED_REFERENCE`)
	if err = repo.EraseDatabase(work, seal, validate); err != nil {
		t.Fatal(err)
	}
	if err = repo.EraseDatabase(work, seal, validate); err != nil {
		t.Fatal("retry", err)
	}
	for _, query := range []string{`SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=42`, `SELECT COUNT(*) FROM WEO_VISIT_DAILY WHERE VD_USR_SEQ=42`, `SELECT COUNT(*) FROM ALUMNI_MESSAGE WHERE AM_SENDER_SEQ=42 OR AM_RECVR_SEQ=42`, `SELECT COUNT(*) FROM WEO_ORDER WHERE O_SEQ=1`, `SELECT COUNT(*) FROM WEO_PG_DATA WHERE O_SEQ=1`, `SELECT COUNT(*) FROM ALUMNI_PROFILE_FILE_HISTORY WHERE USR_SEQ=42`} {
		db.Get(&count, query)
		if count != 0 {
			t.Fatal("personal data survived", query)
		}
	}
	total, donors, err := NewDonationRepository(db).GetReceivedDonationAggregate()
	if err != nil || total != 150 || donors != 2 {
		t.Fatalf("aggregate lost: %d %d %v", total, donors, err)
	}

	// Reviewed historical files join the existing queue after database erasure.
	plan := model.ErasureHistoricalFilePlan{RequestID: receipt.ID, UserSeq: 42, Evidence: "synthetic-inventory-42", OwnershipVerified: true, RetentionRespected: true, Files: []string{"/files/profile/old42.jpg"}}
	beforeFiles, err := repo.ErasureFiles(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.QueueHistoricalErasureFiles(plan, false); err != nil {
		t.Fatal("dry-run", err)
	}
	afterFiles, err := repo.ErasureFiles(receipt.ID)
	if err != nil || len(afterFiles) != len(beforeFiles) {
		t.Fatal("dry-run mutated queue", err)
	}
	wrongPlan := plan
	wrongPlan.UserSeq = 43
	if repo.QueueHistoricalErasureFiles(wrongPlan, true) == nil {
		t.Fatal("wrong request owner accepted")
	}
	db.MustExec(`UPDATE WEO_MEMBER SET USR_PHOTO='https://app.example.org/files/profile/old42.jpg' WHERE USR_SEQ=43`)
	if repo.QueueHistoricalErasureFiles(plan, true) == nil {
		t.Fatal("another member's profile queued")
	}
	db.MustExec(`UPDATE WEO_MEMBER SET USR_PHOTO='' WHERE USR_SEQ=43`)
	db.MustExec(`CREATE TABLE WEO_BOARDBBS (SEQ INT PRIMARY KEY,USR_SEQ INT,CONTENTS TEXT) ENGINE=InnoDB; INSERT INTO WEO_BOARDBBS VALUES (1,43,TO_BASE64('<img src="/old/upload/profile/old42.jpg">'))`)
	if repo.QueueHistoricalErasureFiles(plan, true) == nil {
		t.Fatal("another post's embedded file queued")
	}
	db.MustExec(`DELETE FROM WEO_BOARDBBS`)
	if err = repo.SetErasureMode(receipt.ID, 7, "manual"); err != nil {
		t.Fatal(err)
	}
	if repo.QueueHistoricalErasureFiles(plan, true) == nil {
		t.Fatal("manual takeover ignored by queue")
	}
	if err = repo.SetErasureMode(receipt.ID, 7, "automatic"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = repo.QueueHistoricalErasureFiles(plan, true); err != nil {
			t.Fatal("enqueue/retry", err)
		}
	}
	afterFiles, err = repo.ErasureFiles(receipt.ID)
	if err != nil || len(afterFiles) != len(beforeFiles)+1 {
		t.Fatal("queue was not idempotent", err)
	}
	targets, err = repo.ErasureTargets(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if target.Name == "historical_files" && (target.Verified() || target.Evidence != "reviewed-file-plan:synthetic-inventory-42") {
			t.Fatal("inventory falsely completed target", target)
		}
	}
	if err = repo.FinishAutomaticErasure(work); err == nil {
		t.Fatal("unfinished file work accepted")
	}
	files, err := repo.ErasureFiles(receipt.ID)
	if err != nil || len(files) != len(beforeFiles)+1 {
		t.Fatal(files, err)
	}
	for _, file := range files {
		if err = repo.ErasureFileDone(file.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err = repo.FinishAutomaticErasure(work); err == nil {
		t.Fatal("missing external evidence accepted")
	}
	if _, err = repo.LoadErasureContext(receipt.ID); err != nil {
		t.Fatal("external context lost after member erasure", err)
	}
	if err = repo.RecordExternalErasure(receipt.ID, " "); err == nil {
		t.Fatal("empty external proof accepted")
	}
	if err = repo.RecordExternalErasure(receipt.ID, "synthetic-external-proof"); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.LoadErasureContext(receipt.ID); err != sql.ErrNoRows {
		t.Fatal("verified context retained", err)
	}
	partial, err := repo.Receipt(strings.Repeat("a", 64))
	if err != nil || !partial.DatabaseErased || partial.Status == "completed" {
		t.Fatal("partial erasure misreported", err)
	}
	if err = repo.FinishAutomaticErasure(work); err == nil {
		t.Fatal("active receipt contact work completed deletion")
	}
	if err = repo.ResolveReceiptWork(receipt.ID, 42, model.AccountDeletionResolution{ReceiptWorkStatus: "completed"}); err == nil {
		t.Fatal("self completion allowed")
	}
	if err = repo.ResolveReceiptWork(receipt.ID, 7, model.AccountDeletionResolution{ReceiptWorkStatus: "not_required"}); err == nil {
		t.Fatal("active work dismissed without cleanup")
	}
	if err = repo.ResolveReceiptWork(receipt.ID, 7, model.AccountDeletionResolution{ReceiptWorkStatus: "completed", ResultNotified: true, ContactErased: true, EvidenceReference: "synthetic-delivered-and-contact-cleared"}); err != nil {
		t.Fatal(err)
	}
	if err = repo.FinishAutomaticErasure(work); err != nil {
		t.Fatal(err)
	}
	final, err := repo.Receipt(strings.Repeat("a", 64))
	if err != nil || final.Status != "completed" || final.RetentionUntil == nil {
		t.Fatal(final, err)
	}
	db.Get(&count, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=43`)
	if count != 1 {
		t.Fatal("other member deleted")
	}
	db.Get(&count, `SELECT COUNT(*) FROM ALUMNI_MESSAGE WHERE AM_SEQ=3`)
	if count != 1 {
		t.Fatal("other content deleted")
	}
	db.Get(&count, `SELECT COUNT(*) FROM ALUMNI_DONATION_LEGAL_ARCHIVE`)
	if count != 1 {
		t.Fatal("legal archive missing")
	}
	// Retry must never extend context TTL or revive expired handoff information.
	other, err := repo.Create(43, strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveErasureContext(other.ID, []byte("original-ciphertext")); err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveErasureContext(other.ID, []byte("replacement-ciphertext")); err != nil {
		t.Fatal(err)
	}
	data, err := repo.LoadErasureContext(other.ID)
	if err != nil || string(data) != "original-ciphertext" {
		t.Fatal("context overwritten")
	}
	db.MustExec(`UPDATE ALUMNI_ERASURE_CONTEXT SET EXPIRES_AT=DATE_SUB(UTC_TIMESTAMP(),INTERVAL 1 DAY) WHERE REQUEST_ID=?`, other.ID)
	if err = repo.SaveErasureContext(other.ID, []byte("replacement-ciphertext")); err == nil {
		t.Fatal("expired handoff revived")
	}
	if _, err = repo.LoadErasureContext(other.ID); err != sql.ErrNoRows {
		t.Fatal("expired identifiers readable")
	}
	db.MustExec(`UPDATE ALUMNI_DONATION_LEGAL_ARCHIVE SET RETAIN_UNTIL='2020-01-01'`)
	if err = repo.PurgeExpiredPrivacyRecords(context.Background(), 100); err != nil {
		t.Fatal(err)
	}
	db.Get(&count, `SELECT COUNT(*) FROM ALUMNI_DONATION_LEGAL_ARCHIVE`)
	if count != 0 {
		t.Fatal("expired legal archive survived")
	}
	if err = db.Get(&count, `SELECT COUNT(*) FROM ALUMNI_ERASURE_CONTEXT`); err != nil || count != 0 {
		t.Fatal("expired context survived", err)
	}
}
