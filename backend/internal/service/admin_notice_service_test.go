package service

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

type noticePublishedNotifierSpy struct {
	calls   int
	seq     int
	subject string
}

func (s *noticePublishedNotifierSpy) NotifyNoticePublished(noticeSeq int, subject string) {
	s.calls++
	s.seq = noticeSeq
	s.subject = subject
}

// newAdminNoticeServiceTest wires the service over a mocked database and
// primes the two statements a no-attachment Create issues.
func newAdminNoticeServiceTest(t *testing.T) (*AdminNoticeService, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	wrapped := sqlx.NewDb(db, "sqlmock")
	mock.ExpectQuery(`SELECT MAX\(SEQ\) FROM WEO_BOARDBBS`).
		WillReturnRows(sqlmock.NewRows([]string{"MAX(SEQ)"}).AddRow(500))
	mock.ExpectExec(`(?s)INSERT INTO WEO_BOARDBBS`).WillReturnResult(sqlmock.NewResult(1, 1))
	service := NewAdminNoticeService(repository.NewAdminNoticeRepository(wrapped), repository.NewFileRepository(wrapped))
	return service, mock, func() { _ = db.Close() }
}

func TestAdminNoticeServiceBroadcastsANewlyPublishedNotice(t *testing.T) {
	service, mock, cleanup := newAdminNoticeServiceTest(t)
	defer cleanup()
	spy := &noticePublishedNotifierSpy{}
	service.SetNoticePublishedNotifier(spy)

	seq, err := service.Create("장학금 안내", "본문", "관리자", 7, "N", nil)
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if seq != 501 {
		t.Fatalf("seq = %d", seq)
	}
	if spy.calls != 1 || spy.seq != 501 || spy.subject != "장학금 안내" {
		t.Fatalf("notifier saw %d calls, seq=%d subject=%q", spy.calls, spy.seq, spy.subject)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// The notice row is live the instant the insert commits, so a failure to link
// its attachments must not swallow the announcement — while the caller still
// sees the error.
func TestAdminNoticeServiceBroadcastsEvenWhenAttachmentLinkingFails(t *testing.T) {
	service, mock, cleanup := newAdminNoticeServiceTest(t)
	defer cleanup()
	spy := &noticePublishedNotifierSpy{}
	service.SetNoticePublishedNotifier(spy)
	mock.ExpectExec(`(?s)UPDATE WEO_FILES`).WillReturnError(errors.New("attach failed"))

	seq, err := service.Create("장학금 안내", "본문", "관리자", 7, "N", []int{31})
	if err == nil {
		t.Fatal("the attachment error must still reach the caller")
	}
	if seq != 501 {
		t.Fatalf("seq = %d", seq)
	}
	if spy.calls != 1 || spy.seq != 501 {
		t.Fatalf("notifier saw %d calls, seq=%d", spy.calls, spy.seq)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Push delivery is off by default, so a service with no notifier must publish
// notices exactly as before.
func TestAdminNoticeServiceCreateIsSafeWithoutANotifier(t *testing.T) {
	service, mock, cleanup := newAdminNoticeServiceTest(t)
	defer cleanup()

	if _, err := service.Create("장학금 안내", "본문", "관리자", 7, "N", nil); err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Editing an existing notice is not news: only Create announces.
func TestAdminNoticeServiceDoesNotBroadcastOnUpdateOrPin(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	wrapped := sqlx.NewDb(db, "sqlmock")
	service := NewAdminNoticeService(repository.NewAdminNoticeRepository(wrapped), repository.NewFileRepository(wrapped))
	spy := &noticePublishedNotifierSpy{}
	service.SetNoticePublishedNotifier(spy)

	mock.ExpectExec(`(?s)UPDATE WEO_BOARDBBS`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE WEO_FILES`).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := service.Update(501, "수정된 제목", "본문", "N", nil); err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if spy.calls != 0 {
		t.Fatalf("Update broadcast %d times", spy.calls)
	}
}
