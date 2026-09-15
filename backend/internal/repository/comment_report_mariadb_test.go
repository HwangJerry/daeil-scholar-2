package repository

import (
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"os"
	"sync"
	"testing"
)

func TestCommentReportsMariaDB(t *testing.T) {
	if os.Getenv("COMMENT_REPORT_DOCKER_INTEGRATION") != "1" {
		t.Skip("set COMMENT_REPORT_DOCKER_INTEGRATION=1")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE WEO_MEMBER (USR_SEQ INT PRIMARY KEY,USR_STATUS VARCHAR(3)) ENGINE=InnoDB; INSERT INTO WEO_MEMBER VALUES(7,'CCC'),(8,'BBB'),(42,'CCC');
 CREATE TABLE WEO_BOARDBBS (SEQ INT PRIMARY KEY,GATE VARCHAR(20),OPEN_YN CHAR(1)) ENGINE=InnoDB;
 CREATE TABLE WEO_BOARDCOMAND (SEQ INT PRIMARY KEY,JOIN_SEQ INT,USR_SEQ INT,BC_TYPE CHAR(1),OPEN_YN CHAR(1),CONTENTS TEXT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
 INSERT INTO WEO_BOARDBBS VALUES (1,'NOTICE','Y'),(2,'NOTICE','N'),(3,'OTHER','Y');
 INSERT INTO WEO_BOARDCOMAND VALUES (11,1,42,'B','Y','server evidence'),(12,1,42,'B','N','deleted'),(13,2,42,'B','Y','hidden parent'),(14,3,42,'B','Y','other gate'),(15,1,42,'A','Y','other type'),(16,1,42,'B','Y','race'),(17,1,42,'B','Y','rollback');`)
	migration, err := os.ReadFile("../../migrations/070_create_comment_reports.sql")
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(string(migration))
	db.MustExec(string(migration))
	repo := &CommentReportRepository{DB: db}
	request := model.CommentReportRequest{PostID: 1, CommentID: 11, Reason: "spam", Details: "details"}
	for _, test := range []struct {
		reporter      int
		post, comment int64
	}{{42, 1, 11}, {7, 1, 12}, {7, 2, 13}, {7, 3, 14}, {7, 1, 15}, {7, 2, 11}, {7, 1, 999}} {
		req := request
		req.PostID = test.post
		req.CommentID = test.comment
		if _, err := repo.Create(test.reporter, req); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("unauthorized target %+v: %v", test, err)
		}
	}
	const concurrent = 8
	ids := make(chan int64, concurrent)
	failures := make(chan error, concurrent)
	var wg sync.WaitGroup
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); id, err := repo.Create(7, request); ids <- id; failures <- err }()
	}
	wg.Wait()
	close(ids)
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	var reportID int64
	for id := range ids {
		if reportID != 0 && reportID != id {
			t.Fatal("duplicate report IDs")
		}
		reportID = id
	}
	second, err := repo.Create(8, request)
	if err != nil {
		t.Fatal(err)
	}
	list, err := repo.List("open", 0)
	if err != nil || len(list) != 2 || list[0].Content != "server evidence" || !list[0].Visible {
		t.Fatalf("list=%+v err=%v", list, err)
	}
	next, err := repo.List("open", second)
	if err != nil || len(next) != 1 || next[0].ID != reportID {
		t.Fatalf("cursor=%+v %v", next, err)
	}
	decision := model.CommentReportResolution{Status: "removed", Note: "policy violation"}
	if err := repo.Resolve(reportID, 99, decision); err != nil {
		t.Fatal(err)
	}
	if err := repo.Resolve(reportID, 99, decision); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if err := repo.Resolve(second, 99, model.CommentReportResolution{Status: "dismissed", Note: "different"}); !errors.Is(err, ErrCommentReportResolved) {
		t.Fatalf("conflict: %v", err)
	}
	list, err = repo.List("removed", 0)
	if err != nil || len(list) != 2 || list[0].Visible {
		t.Fatalf("removed=%+v %v", list, err)
	}
	request.Details = "must not overwrite"
	if id, err := repo.Create(7, request); err != nil || id != reportID {
		t.Fatalf("retry after hide: %v", err)
	}
	var details string
	db.Get(&details, "SELECT DETAILS FROM ALUMNI_COMMENT_REPORT WHERE REPORT_ID=?", reportID)
	if details != "details" {
		t.Fatal("snapshot overwritten")
	}
	// Two different reports on the same comment serialize even when decisions race.
	request.CommentID = 16
	a, err := repo.Create(7, request)
	if err != nil {
		t.Fatal(err)
	}
	b, err := repo.Create(8, request)
	if err != nil {
		t.Fatal(err)
	}
	failures = make(chan error, 2)
	go func() { failures <- repo.Resolve(a, 99, decision) }()
	go func() { failures <- repo.Resolve(b, 99, decision) }()
	for i := 0; i < 2; i++ {
		if err := <-failures; err != nil {
			t.Fatal(err)
		}
	}
	// Simulate an audit-write error: hiding must roll back with the report update.
	request.CommentID = 17
	id, err := repo.Create(7, request)
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(`CREATE TRIGGER reject_report_update BEFORE UPDATE ON ALUMNI_COMMENT_REPORT FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='test audit failure'`)
	if err := repo.Resolve(id, 99, decision); err == nil {
		t.Fatal("audit failure accepted")
	}
	var visible string
	db.Get(&visible, "SELECT OPEN_YN FROM WEO_BOARDCOMAND WHERE SEQ=17")
	if visible != "Y" {
		t.Fatal("comment removal did not roll back")
	}
	db.MustExec(`DROP TRIGGER reject_report_update`)
	// The erasure plan removes restricted evidence and both member identities.
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	schema, err := readErasureSchema(tx)
	if err != nil {
		t.Fatal(err)
	}
	if err := eraseAccountReferences(tx, schema, 7, ""); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var count int
	db.Get(&count, "SELECT COUNT(*) FROM ALUMNI_COMMENT_REPORT WHERE REPORTER_SEQ=7 OR REPORTED_SEQ=7")
	if count != 0 {
		t.Fatal("erasure retained report")
	}
}
