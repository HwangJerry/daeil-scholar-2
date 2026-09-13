// account_erasure_preview_mariadb_test.go — The preview equals the real erasure; approval binds to the reviewed plan.
package repository

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func TestErasurePreviewMatchesErasureOnMariaDB101(t *testing.T) {
	if os.Getenv("AUTOMATIC_ERASURE_DOCKER_INTEGRATION") != "1" {
		t.Skip("set AUTOMATIC_ERASURE_DOCKER_INTEGRATION=1")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE WEO_MEMBER (USR_SEQ INT PRIMARY KEY,USR_STATUS CHAR(3),USR_ID VARCHAR(50),USR_NAME VARCHAR(100),USR_EMAIL VARCHAR(100),USR_PHONE VARCHAR(32),USR_PASS VARCHAR(100),USR_PHOTO VARCHAR(200),USR_BIZ_CARD VARCHAR(200),USR_THUMNAIL TEXT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE WEO_MEMBER_SOCIAL (USR_SEQ INT,NMS_GATE CHAR(2),NMS_ID VARCHAR(100)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE ALUMNI_ADMIN_ROLE (USR_SEQ INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE ALUMNI_SOCIAL_REVOCATION_OUTBOX (ID BIGINT AUTO_INCREMENT PRIMARY KEY,USR_SEQ INT,PROVIDER CHAR(2),ACTION VARCHAR(30),STATUS VARCHAR(30),NEXT_ATTEMPT_AT DATETIME,CREATED_AT DATETIME,UPDATED_AT DATETIME) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE ALUMNI_MESSAGE (AM_SEQ INT PRIMARY KEY,AM_SENDER_SEQ INT,AM_RECVR_SEQ INT,AM_CONTENT TEXT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE ALUMNI_NOTIFICATION (AN_SEQ INT PRIMARY KEY,USR_SEQ INT,AN_TYPE VARCHAR(20),AN_REF_SEQ INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE ALUMNI_PUSH_OUTBOX (ID INT PRIMARY KEY,USR_SEQ INT,EVENT_TYPE VARCHAR(20),EVENT_ID INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE WEO_VISIT_DAILY (VD_USR_SEQ INT,VD_VISITOR_ID VARCHAR(50)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE WEO_BOARDBBS (SEQ INT PRIMARY KEY,USR_SEQ INT,SUBJECT VARCHAR(200),CONTENTS TEXT,USR_NAME VARCHAR(100),PASSWORD VARCHAR(100)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE WEO_BOARDCOMAND (SEQ INT PRIMARY KEY,USR_SEQ INT,JOIN_SEQ INT,CONTENTS TEXT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE WEO_FILES (F_SEQ INT PRIMARY KEY,F_GATE CHAR(2),F_JOIN_SEQ INT,FILE_PATH VARCHAR(200),FILE_NAME VARCHAR(100)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE WEO_ORDER (O_SEQ INT PRIMARY KEY,USR_SEQ INT,O_ACCOUNT_USR_SEQ INT,O_DONOR_NAME VARCHAR(100),O_DONOR_PHONE VARCHAR(32),O_DONATION_DATE DATE,O_GROSS_AMOUNT BIGINT,O_REFUNDED_AMOUNT BIGINT,O_NET_RECEIVED_AMOUNT BIGINT,O_SOURCE VARCHAR(30),O_TRANSACTION_NO VARCHAR(100),O_LIFECYCLE_STATUS VARCHAR(30),O_TYPE CHAR(1)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE WEO_PG_DATA (O_SEQ INT,NUM_CARD VARCHAR(50)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
INSERT INTO WEO_MEMBER VALUES (42,'CCC','fake42','Synthetic Member','fake42@example.org','01000000042','hash42','/uploads/profile/42.jpg','',NULL),(43,'CCC','fake43','Other Member','fake43@example.org','01000000043','hash43','','',NULL);
INSERT INTO WEO_MEMBER_SOCIAL VALUES (42,'KT','synthetic-kakao');
INSERT INTO ALUMNI_MESSAGE VALUES (1,42,43,'outbound private'),(2,43,42,'inbound private'),(3,43,43,'unrelated');
INSERT INTO ALUMNI_NOTIFICATION VALUES (1,43,'message',1),(2,42,'like',0),(3,43,'like',0);
INSERT INTO ALUMNI_PUSH_OUTBOX VALUES (1,42,'message',2),(2,43,'message',3);
INSERT INTO WEO_VISIT_DAILY VALUES (42,'visitor42'),(43,'visitor43');
INSERT INTO WEO_BOARDBBS VALUES (10,42,'withdrawn subject','withdrawn body','Synthetic Member','postpw'),(11,43,'kept subject','kept body','Other Member','otherpw');
INSERT INTO WEO_BOARDCOMAND VALUES (20,42,11,'withdrawn reply'),(21,43,10,'kept reply');
INSERT INTO WEO_FILES VALUES (5,'PF',42,'/uploads/profile','42.jpg'),(6,'PF',43,'/uploads/profile','43.jpg');
INSERT INTO WEO_ORDER VALUES (1,42,42,'Synthetic Member','01000000042','2025-01-01',100,0,100,'direct','fake-tx-1','completed','A'),(2,43,43,'Other Member','01000000043','2025-01-01',50,0,50,'direct','fake-tx-2','completed','A');
INSERT INTO WEO_PG_DATA VALUES (1,'fake-card'),(2,'other-card');`)
	for _, name := range []string{"055_create_message_reports.sql", "056_create_account_deletion_requests.sql", "057_create_automatic_account_erasure.sql", "066_schedule_account_erasure.sql", "067_cancel_account_deletion.sql", "059_create_erasure_context.sql", "060_create_erasure_targets.sql", "061_create_erasure_receipt_work.sql", "062_create_profile_file_history.sql", "063_bind_donation_retention_source.sql"} {
		data, err := os.ReadFile("../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(data)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	db.MustExec(`INSERT INTO ALUMNI_UPLOAD_OWNER (F_SEQ,USR_SEQ,URL_PATH) VALUES (5,42,'/uploads/profile/42.jpg')`)
	repo := &AccountDeletionRequestRepository{WaitHours: 72, DB: db}
	receipt, err := repo.Create(42, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}

	// The preview never writes and reports the worker's own hold conditions.
	before := snapshotErasureTables(t, db)
	preview, err := repo.PreviewErasure(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, snapshotErasureTables(t, db)) {
		t.Fatal("preview changed stored data")
	}
	for _, code := range []string{"DONATION_RETENTION_REVIEW_REQUIRED", "RECEIPT_CONTACT_REVIEW_REQUIRED"} {
		if !containsString(preview.Blockers, code) {
			t.Fatalf("blocker %s missing: %v", code, preview.Blockers)
		}
	}
	if err = repo.ResolveReceiptWork(receipt.ID, 7, model.AccountDeletionResolution{ReceiptWorkStatus: "not_required", EvidenceReference: "synthetic-review"}); err != nil {
		t.Fatal(err)
	}
	source, err := repo.DonationRetentionSource(1)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveDonationRetention(model.DonationRetentionDecision{OrderID: 1, Basis: "erase", SourceFingerprint: source, Evidence: "synthetic-review"}); err != nil {
		t.Fatal(err)
	}
	if err = repo.SetErasureMode(receipt.ID, 7, "manual"); err != nil {
		t.Fatal(err)
	}
	preview, err = repo.PreviewErasure(receipt.ID)
	if err != nil || len(preview.Blockers) != 0 || len(preview.Unhandled) != 0 {
		t.Fatalf("unexpected hold: %v %v %v", err, preview.Blockers, preview.Unhandled)
	}
	again, err := repo.PreviewErasure(receipt.ID)
	if err != nil || again.PlanDigest != preview.PlanDigest {
		t.Fatal("digest is not deterministic", err)
	}
	counts := map[string]int64{}
	for _, table := range preview.Tables {
		counts[table.Table] = table.Count
	}
	wantCounts := map[string]int64{"WEO_FILES": 1, "WEO_PG_DATA": 1, "WEO_ORDER": 1, "ALUMNI_DONATION_RETENTION": 1, "WEO_BOARDBBS": 1,
		"ALUMNI_PUSH_OUTBOX": 1, "ALUMNI_NOTIFICATION": 2, "WEO_BOARDCOMAND": 1, "ALUMNI_MESSAGE": 2, "WEO_VISIT_DAILY": 1,
		"WEO_MEMBER_SOCIAL": 1, "ALUMNI_UPLOAD_OWNER": 1, "ALUMNI_PROFILE_FILE_HISTORY": 1, "WEO_MEMBER": 1}
	if !reflect.DeepEqual(counts, wantCounts) {
		t.Fatalf("counts = %v, want %v", counts, wantCounts)
	}
	// Migration 062 backfills the current photo into ALUMNI_PROFILE_FILE_HISTORY.
	// Secrets and private message bodies never leave the server.
	encoded, _ := json.Marshal(preview)
	for _, secret := range []string{"hash42", "postpw", "outbound private", "inbound private"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("preview exposed %q", secret)
		}
	}
	assertAnonymizedPost(t, preview)

	// Apply the real erasure steps in EraseDatabase order and compare row sets.
	w := model.ErasureWork{RequestID: receipt.ID, UserSeq: 42}
	subject, err := repo.ErasureExternalSubject(w)
	if err != nil {
		t.Fatal(err)
	}
	w.ContextRetentions = subject.Retentions
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	_, affected, err := repo.collectErasurePreview(tx, receipt.ID, 42)
	if err != nil {
		t.Fatal(err)
	}
	beforeRows := snapshotErasureTables(t, tx)
	s, err := readErasureSchema(tx)
	if err != nil {
		t.Fatal(err)
	}
	accept := func(model.DonationRetentionDecision) error { return nil }
	if err = queueErasureFiles(tx, s, w, repo.SiteOrigin); err != nil {
		t.Fatal(err)
	}
	if err = eraseDonations(tx, s, w, func(b []byte) ([]byte, error) { return b, nil }, accept); err != nil {
		t.Fatal(err)
	}
	if err = erasePostChildren(tx, s, 42); err != nil {
		t.Fatal(err)
	}
	if err = eraseAccountReferences(tx, s, 42, "fake42@example.org"); err != nil {
		t.Fatal(err)
	}
	if err = s.erase(tx, "WEO_MEMBER", "USR_SEQ=?", 42); err != nil {
		t.Fatal(err)
	}
	afterRows := snapshotErasureTables(t, tx)
	planned := map[string]previewTable{}
	for _, table := range affected {
		planned[table.Table] = table
	}
	bookkeeping := map[string]bool{"ALUMNI_ACCOUNT_DELETION_REQUEST": true, "ALUMNI_ERASED_DONATION_TOTAL": true, "ALUMNI_ERASURE_FILE": true}
	for table, rows := range beforeRows {
		removed, added := multisetDiff(rows, afterRows[table]), multisetDiff(afterRows[table], rows)
		plan, ok := planned[table]
		switch {
		case ok:
			want := append([]string(nil), plan.rowHashes...)
			sort.Strings(want)
			if !reflect.DeepEqual(removed, want) {
				t.Fatalf("%s: erased rows differ from preview (%d vs %d)", table, len(removed), len(want))
			}
			if (plan.Action == model.ErasureActionAnonymize) != (len(added) == len(removed)) || (plan.Action == model.ErasureActionDelete && len(added) != 0) {
				t.Fatalf("%s: action %s does not match change", table, plan.Action)
			}
		case table == "ALUMNI_ERASURE_FILE":
			if len(removed) != 0 || len(added) != len(preview.Files) {
				t.Fatalf("queued files differ from preview: %d vs %d", len(added), len(preview.Files))
			}
		case !bookkeeping[table] && (len(removed) != 0 || len(added) != 0):
			t.Fatalf("%s changed but was not in the preview", table)
		}
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	// Approval is bound to the exact reviewed plan.
	if err = repo.ExpediteReviewed(receipt.ID, 7, strings.Repeat("0", 64)); !errors.Is(err, ErrErasurePlanChanged) {
		t.Fatal("unreviewed digest accepted", err)
	}
	db.MustExec(`INSERT INTO ALUMNI_NOTIFICATION VALUES (4,42,'like',0)`)
	if err = repo.ExpediteReviewed(receipt.ID, 7, preview.PlanDigest); !errors.Is(err, ErrErasurePlanChanged) {
		t.Fatal("changed records accepted without re-review", err)
	}
	var expedited int
	if err = db.Get(&expedited, `SELECT COUNT(*) FROM ALUMNI_ERASURE_SCHEDULE WHERE REQUEST_ID=? AND EXPEDITED_AT IS NOT NULL`, receipt.ID); err != nil || expedited != 0 {
		t.Fatal("rejected approval scheduled processing", err)
	}
	fresh, err := repo.PreviewErasure(receipt.ID)
	if err != nil || fresh.PlanDigest == preview.PlanDigest {
		t.Fatal("new record did not change the plan", err)
	}
	if err = repo.ExpediteReviewed(receipt.ID, 42, fresh.PlanDigest); err == nil {
		t.Fatal("member approved own erasure")
	}
	if err = repo.ExpediteReviewed(receipt.ID, 7, fresh.PlanDigest); err != nil {
		t.Fatal(err)
	}
	var state struct {
		By   int    `db:"EXPEDITED_BY"`
		Mode string `db:"MODE"`
	}
	if err = db.Get(&state, `SELECT s.EXPEDITED_BY,e.MODE FROM ALUMNI_ERASURE_SCHEDULE s JOIN ALUMNI_ACCOUNT_ERASURE e ON e.REQUEST_ID=s.REQUEST_ID WHERE s.REQUEST_ID=? AND s.EXPEDITED_AT IS NOT NULL`, receipt.ID); err != nil || state.By != 7 || state.Mode != "automatic" {
		t.Fatalf("approved processing not started: %+v %v", state, err)
	}
}

func assertAnonymizedPost(t *testing.T, preview model.ErasurePreview) {
	t.Helper()
	for _, table := range preview.Tables {
		if table.Table != "WEO_BOARDBBS" {
			continue
		}
		row := table.Rows[0]
		for i, column := range table.Columns {
			switch column {
			case "SUBJECT", "CONTENTS":
				if row.After[i] == nil || *row.After[i] != erasedPostText {
					t.Fatalf("%s after = %v", column, row.After[i])
				}
			case "USR_SEQ":
				if *row.Before[i] != "42" || *row.After[i] != "0" {
					t.Fatal("author not detached")
				}
			}
		}
		return
	}
	t.Fatal("post anonymization missing from preview")
}

func snapshotErasureTables(t *testing.T, q sqlx.Queryer) map[string][]string {
	t.Helper()
	var tables []string
	if err := sqlx.Select(q, &tables, `SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_TYPE='BASE TABLE'`); err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, table := range tables {
		rows, err := q.Queryx("SELECT * FROM `" + table + "`")
		if err != nil {
			t.Fatal(err)
		}
		out[table] = []string{}
		for rows.Next() {
			values, err := rows.SliceScan()
			if err != nil {
				t.Fatal(err)
			}
			raw := make([]*string, len(values))
			for i, value := range values {
				raw[i] = previewRawValue(value)
			}
			out[table] = append(out[table], previewRowHash(raw))
		}
		rows.Close()
		sort.Strings(out[table])
	}
	return out
}

// multisetDiff returns sorted elements of a missing from b, counting duplicates.
func multisetDiff(a, b []string) []string {
	remaining := map[string]int{}
	for _, v := range b {
		remaining[v]++
	}
	out := []string{}
	for _, v := range a {
		if remaining[v] > 0 {
			remaining[v]--
			continue
		}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
