package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func newNotificationInboxRepositoryTest(t *testing.T) (*NotificationInboxRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewNotificationInboxRepository(sqlx.NewDb(db, "sqlmock")), mock
}

// The first page has no SEQ bound and asks for one extra row to detect more pages.
func TestNotificationInboxRepositoryListsPublishedNoticesInsideWindow(t *testing.T) {
	repo, mock := newNotificationInboxRepositoryTest(t)
	mock.ExpectQuery(`(?s)FROM WEO_BOARDBBS WHERE GATE = 'NOTICE' AND OPEN_YN = 'Y'.*REG_DATE >= NOW\(\) - INTERVAL \? DAY ORDER BY SEQ DESC LIMIT \?`).
		WithArgs(NotificationInboxWindowDays, 21).
		WillReturnRows(sqlmock.NewRows([]string{"SEQ", "SUBJECT", "REG_DATE"}))

	rows, err := repo.ListNoticeNotifications(0, 20)
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("rows = %#v, err = %v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationInboxRepositoryPagesBeforeCursor(t *testing.T) {
	repo, mock := newNotificationInboxRepositoryTest(t)
	mock.ExpectQuery(`(?s)INTERVAL \? DAY AND SEQ < \? ORDER BY SEQ DESC LIMIT \?`).
		WithArgs(NotificationInboxWindowDays, 500, 11).
		WillReturnRows(sqlmock.NewRows([]string{"SEQ", "SUBJECT", "REG_DATE"}))

	if _, err := repo.ListNoticeNotifications(500, 10); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationInboxRepositoryReportsNeverSeenAsZero(t *testing.T) {
	repo, mock := newNotificationInboxRepositoryTest(t)
	mock.ExpectQuery(`SELECT LAST_SEEN_POST_SEQ FROM ALUMNI_NOTIFICATION_INBOX_STATE WHERE USR_SEQ = \?`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"LAST_SEEN_POST_SEQ"}))

	lastSeen, err := repo.GetLastSeenPostSeq(42)
	if err != nil || lastSeen != 0 {
		t.Fatalf("lastSeen = %d, err = %v", lastSeen, err)
	}
}

func TestNotificationInboxRepositoryCountsBySeqMarker(t *testing.T) {
	repo, mock := newNotificationInboxRepositoryTest(t)
	mock.ExpectQuery(`(?s)GATE = 'NOTICE' AND OPEN_YN = 'Y'.*INTERVAL \? DAY.*SEQ > COALESCE\(.*SELECT LAST_SEEN_POST_SEQ FROM ALUMNI_NOTIFICATION_INBOX_STATE WHERE USR_SEQ = \?.*0\)`).
		WithArgs(NotificationInboxWindowDays, 42).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(3))

	count, err := repo.CountUnseenNotices(42)
	if err != nil || count != 3 {
		t.Fatalf("count = %d, err = %v", count, err)
	}
}

// The marker is clamped to the newest notice SEQ before the forward-only upsert.
func TestNotificationInboxRepositoryClampsAndOnlyMovesSeenForward(t *testing.T) {
	repo, mock := newNotificationInboxRepositoryTest(t)
	mock.ExpectQuery(`SELECT COALESCE\(MAX\(SEQ\), 0\) FROM WEO_BOARDBBS WHERE GATE = 'NOTICE'`).
		WillReturnRows(sqlmock.NewRows([]string{"MAX"}).AddRow(510))
	mock.ExpectExec(`(?s)INSERT INTO ALUMNI_NOTIFICATION_INBOX_STATE \(USR_SEQ, LAST_SEEN_POST_SEQ, UPD_DATE\).*VALUES \(\?, \?, NOW\(\)\).*ON DUPLICATE KEY UPDATE.*LAST_SEEN_POST_SEQ = GREATEST\(LAST_SEEN_POST_SEQ, VALUES\(LAST_SEEN_POST_SEQ\)\)`).
		WithArgs(42, 510).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkSeenThrough(42, 999999); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
