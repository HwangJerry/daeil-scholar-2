package service

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

type capturedCommentTime struct{ value string }

func (c *capturedCommentTime) Match(value driver.Value) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", s, commentKST)
	if err != nil || time.Since(parsed) < -time.Second || time.Since(parsed) > 5*time.Second {
		return false
	}
	c.value = s
	return true
}

func TestCommentCreateAndListShareKSTTimestamp(t *testing.T) {
	// A UTC host must still persist Korean wall time and return the same minute.
	previous := time.Local
	time.Local = time.UTC
	defer func() { time.Local = previous }()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := NewCommentService(repository.NewCommentRepository(sqlx.NewDb(db, "sqlmock")))
	stamp := &capturedCommentTime{}
	mock.ExpectExec("INSERT INTO WEO_BOARDCOMAND").WithArgs(42, 7, "테스트", "댓글", stamp).WillReturnResult(sqlmock.NewResult(99, 1))
	created, err := svc.AddComment(42, 7, "테스트", "댓글")
	if err != nil {
		t.Fatal(err)
	}
	expected := stamp.value[:16]
	if created.RegDate != expected {
		t.Fatalf("created timestamp = %q, want %q", created.RegDate, expected)
	}
	encoded, err := json.Marshal(created)
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]interface{}
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatal(err)
	}
	if response["regDate"] != expected {
		t.Fatalf("missing JSON timestamp: %s", encoded)
	}
	mock.ExpectQuery("(?s)DATE_FORMAT\\(REG_DATE, '%Y-%m-%d %H:%i'\\) AS REG_DATE.*FROM WEO_BOARDCOMAND").WithArgs(42).WillReturnRows(sqlmock.NewRows([]string{"BC_SEQ", "JOIN_SEQ", "USR_SEQ", "NICKNAME", "CONTENTS", "REG_DATE"}).AddRow(99, 42, 7, "테스트", "댓글", expected))
	comments, err := svc.GetComments(42)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 || comments[0].RegDate != created.RegDate {
		t.Fatalf("list differs: %#v", comments)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCommentTimestampKSTDateRollover(t *testing.T) {
	instant := time.Date(2026, 9, 10, 15, 5, 59, 0, time.UTC)
	if got := instant.In(commentKST).Format(commentTimestampLayout); got != "2026-09-11 00:05" {
		t.Fatal(got)
	}
}
