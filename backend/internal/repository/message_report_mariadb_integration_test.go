package repository

import (
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"os"
	"testing"
)

func TestMessageReportLifecycleOnMariaDB101(t *testing.T) {
	if os.Getenv("MESSAGE_REPORT_DOCKER_INTEGRATION") != "1" {
		t.Skip("set MESSAGE_REPORT_DOCKER_INTEGRATION=1 for the pinned MariaDB integration")
	}
	db := startPasswordResetMariaDB101(t)
	_, err := db.Exec(`CREATE TABLE ALUMNI_MESSAGE (
		AM_SEQ BIGINT PRIMARY KEY, AM_SENDER_SEQ INT NOT NULL, AM_RECVR_SEQ INT NOT NULL,
		AM_CONTENT TEXT NOT NULL, AM_VISIBLE_RECVR CHAR(1) NOT NULL, AM_DEL_RECVR CHAR(1) NOT NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	INSERT INTO ALUMNI_MESSAGE VALUES (91,43,42,'original evidence','Y','N'),
	(92,43,42,'suppressed message','N','N'), (93,43,42,'deleted message','Y','Y');`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/055_create_message_reports.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	// Reapplying the migration must preserve existing reports.
	if _, err = db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	repo := &MessageReportRepository{DB: db}
	request := model.MessageReportRequest{MessageID: 91, Reason: "harassment", Details: "please review"}
	for _, reporter := range []int{43, 99} {
		if _, err := repo.Create(reporter, request); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("unauthorized reporter %d: %v", reporter, err)
		}
	}
	for _, messageID := range []int64{92, 93, 999} {
		if _, err := repo.Create(42, model.MessageReportRequest{MessageID: messageID, Reason: "spam"}); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("unavailable message %d: %v", messageID, err)
		}
	}
	id, err := repo.Create(42, request)
	if err != nil || id <= 0 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	request.Details = "attempt to replace evidence"
	retryID, err := repo.Create(42, request)
	if err != nil || retryID != id {
		t.Fatalf("duplicate id=%d err=%v", retryID, err)
	}
	reports, err := repo.List("open", 0)
	if err != nil || len(reports) != 1 {
		t.Fatalf("reports=%v err=%v", reports, err)
	}
	if reports[0].Content != "original evidence" || reports[0].Details != "please review" {
		t.Fatal("evidence was replaced")
	}
	if err := repo.Resolve(id, 7, model.MessageReportResolution{Status: "removed", Note: "reviewed"}); err != nil {
		t.Fatal(err)
	}
	var content string
	if err := db.Get(&content, "SELECT AM_CONTENT FROM ALUMNI_MESSAGE WHERE AM_SEQ=91"); err != nil {
		t.Fatal(err)
	}
	if content == "original evidence" {
		t.Fatal("offending content still visible")
	}
	reports, err = repo.List("removed", 0)
	if err != nil || len(reports) != 1 || reports[0].Content != "original evidence" {
		t.Fatal("audit snapshot missing")
	}
	if err := repo.Resolve(id, 8, model.MessageReportResolution{Status: "dismissed", Note: "overwrite"}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("resolved report overwritten: %v", err)
	}
}
