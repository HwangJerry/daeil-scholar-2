package repository

import (
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
	"testing"
)

func TestMessageReportOnlyCapturesRecipientsVisibleMessage(t *testing.T) {
	for _, allowed := range []bool{true, false} {
		t.Run(map[bool]string{true: "recipient", false: "unrelated user"}[allowed], func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := &MessageReportRepository{DB: sqlx.NewDb(db, "sqlmock")}
			id := int64(0)
			if allowed {
				id = 31
			}
			mock.ExpectExec(`INSERT INTO ALUMNI_MESSAGE_REPORT[\s\S]*SELECT AM_SEQ, \?, AM_SENDER_SEQ, \?, \?, AM_CONTENT[\s\S]*AM_SEQ = \? AND AM_RECVR_SEQ = \?[\s\S]*AM_SENDER_SEQ <> \? AND AM_VISIBLE_RECVR = 'Y' AND AM_DEL_RECVR = 'N'[\s\S]*ON DUPLICATE KEY UPDATE REPORT_ID = LAST_INSERT_ID\(REPORT_ID\)`).
				WithArgs(42, "spam", "details", 91, 42, 42).WillReturnResult(sqlmock.NewResult(id, 1))
			got, err := repo.Create(42, model.MessageReportRequest{MessageID: 91, Reason: "spam", Details: "details"})
			if allowed && (err != nil || got != 31) {
				t.Fatalf("id=%d err=%v", got, err)
			}
			if !allowed && !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("unauthorized evidence access must fail: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMessageReportResolutionRollsBackContentRemovalIfAuditWriteFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := &MessageReportRepository{DB: sqlx.NewDb(db, "sqlmock")}
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT MESSAGE_ID[\s\S]*STATUS = 'open' FOR UPDATE`).WithArgs(31).
		WillReturnRows(sqlmock.NewRows([]string{"MESSAGE_ID"}).AddRow(91))
	mock.ExpectExec(`UPDATE ALUMNI_MESSAGE SET AM_CONTENT`).WithArgs("[운영정책 위반으로 삭제된 메시지입니다.]", 91).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_MESSAGE_REPORT`).WithArgs("removed", 7, "위협 메시지", 31).WillReturnError(errors.New("write failed"))
	mock.ExpectRollback()
	if err := repo.Resolve(31, 7, model.MessageReportResolution{Status: "removed", Note: "위협 메시지"}); err == nil {
		t.Fatal("must fail atomically")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
