// message_receiving_test.go — SQL recipient filtering and final insert-time receiving guard.
package repository

import (
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
	"strings"
	"testing"
)

func TestRecipientSearchFiltersBeforePaginationWithoutHidingAlumni(t *testing.T) {
	normal, _ := buildAlumniFilters(model.AlumniSearchParams{Name: "예시"})
	recipients, args := buildAlumniFilters(model.AlumniSearchParams{Name: "예시", MessageRecipientsOnly: true})
	if strings.Contains(normal, "USR_MESSAGE_ALLOWED") || !strings.Contains(recipients, "m.USR_MESSAGE_ALLOWED = 'Y'") || len(args) != 1 {
		t.Fatalf("normal=%s recipients=%s", normal, recipients)
	}
}

func TestAcceptMessageRejectsInsertTimeOptOut(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewMessageRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectExec(`INSERT INTO ALUMNI_MESSAGE[\s\S]*WHERE EXISTS \(SELECT 1 FROM WEO_MEMBER recipient WHERE recipient.USR_SEQ = \? AND recipient.USR_MESSAGE_ALLOWED = 'Y'\)`).
		WithArgs(1, 2, "new", "안녕하세요", 2, 1, 2).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT AM_SEQ[\s\S]*WHERE AM_SENDER_SEQ = \? AND AM_CLIENT_MESSAGE_ID = \?`).WithArgs(1, "new").WillReturnError(sql.ErrNoRows)
	_, err = repo.AcceptMessage(1, 2, "new", "안녕하세요")
	var rejection *model.MessageSendRejection
	if !errors.As(err, &rejection) || rejection.Code != model.MessageSendReceivingDisabled {
		t.Fatalf("error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
