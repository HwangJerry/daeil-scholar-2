package repository

import (
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"github.com/jmoiron/sqlx"
)

// TestNotificationInboxOnMariaDB101 runs the inbox queries against the production
// baseline plus migrations 078 to 080 (notice inserts write FEED_CATEGORY_SEQ and OFFICIAL_PROFILE_YN): the 90-day window, the published filter, keyset
// paging and the forward-only, clamped last-seen upsert all have to hold on the real 10.1 engine.
func TestNotificationInboxOnMariaDB101(t *testing.T) {
	inputs := append(mariadb.ProdBaseline(t),
		mariadb.File("../../migrations/078_create_notification_inbox_state.sql"),
		mariadb.File("../../migrations/079_create_feed_categories.sql"),
		mariadb.File("../../migrations/080_add_board_official_profile.sql"))
	db := mariadb.Start(t).NewDatabase(t, inputs...).DB
	notices := NewAdminNoticeRepository(db)
	insert := func(subject string) int {
		t.Helper()
		seq, err := notices.InsertNotice(&model.AdminNoticeInsert{Subject: subject, IsPinned: "N", RegName: "운영자", USRSeq: 1})
		if err != nil {
			t.Fatal(err)
		}
		return seq
	}
	expired, recent, closed, otherGate, older, newest := insert("만료"), insert("최근"), insert("비공개"), insert("다른 게시판"), insert("이전"), insert("최신")
	db.MustExec(`UPDATE WEO_BOARDBBS SET REG_DATE = NOW() - INTERVAL 91 DAY WHERE SEQ = ?`, expired)
	db.MustExec(`UPDATE WEO_BOARDBBS SET REG_DATE = NOW() - INTERVAL 89 DAY WHERE SEQ = ?`, recent)
	db.MustExec(`UPDATE WEO_BOARDBBS SET OPEN_YN = 'N' WHERE SEQ = ?`, closed)
	db.MustExec(`UPDATE WEO_BOARDBBS SET GATE = 'FREE' WHERE SEQ = ?`, otherGate)
	db.MustExec(`UPDATE WEO_BOARDBBS SET REG_DATE = NOW() - INTERVAL 2 DAY WHERE SEQ = ?`, older)

	repo := NewNotificationInboxRepository(db)
	assertInboxSeqs(t, repo, 0, 10, newest, older, recent)
	assertInboxSeqs(t, repo, 0, 1, newest, older)
	assertInboxSeqs(t, repo, older, 10, recent)

	assertInboxUnseen(t, repo, 42, 3)
	if lastSeen, err := repo.GetLastSeenPostSeq(42); err != nil || lastSeen != 0 {
		t.Fatalf("never-seen member lastSeen = %d, err = %v", lastSeen, err)
	}
	// Seen through "older": only the newer notice stays unread.
	markInboxSeen(t, repo, 42, older)
	assertInboxUnseen(t, repo, 42, 1)
	assertInboxLastSeen(t, repo, 42, older)
	assertInboxUnseen(t, repo, 43, 3)

	// An older or retried request never moves the marker back (ON DUPLICATE KEY branch).
	markInboxSeen(t, repo, 42, recent)
	assertInboxLastSeen(t, repo, 42, older)
	assertInboxUnseen(t, repo, 42, 1)

	// A bogus large value is clamped to the newest notice SEQ, so a notice
	// published afterwards is still unread.
	markInboxSeen(t, repo, 42, 1<<30)
	assertInboxLastSeen(t, repo, 42, newest)
	assertInboxUnseen(t, repo, 42, 0)
	later := insert("나중")
	assertInboxUnseen(t, repo, 42, 1)
	markInboxSeen(t, repo, 42, later)
	assertInboxUnseen(t, repo, 42, 0)
	assertInboxStateRows(t, db, 1)
}

func markInboxSeen(t *testing.T, repo *NotificationInboxRepository, userSeq, through int) {
	t.Helper()
	if err := repo.MarkSeenThrough(userSeq, through); err != nil {
		t.Fatal(err)
	}
}

func assertInboxLastSeen(t *testing.T, repo *NotificationInboxRepository, userSeq, want int) {
	t.Helper()
	got, err := repo.GetLastSeenPostSeq(userSeq)
	if err != nil || got != want {
		t.Fatalf("user %d lastSeen = %d, want %d (err %v)", userSeq, got, want, err)
	}
}

func assertInboxSeqs(t *testing.T, repo *NotificationInboxRepository, beforeSeq, size int, want ...int) {
	t.Helper()
	rows, err := repo.ListNoticeNotifications(beforeSeq, size)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.Subject == "" || row.RegDate.IsZero() {
			t.Fatalf("row must carry subject and registration time: %#v", row)
		}
		got = append(got, row.SEQ)
	}
	if len(got) != len(want) {
		t.Fatalf("before %d size %d: seqs = %v, want %v", beforeSeq, size, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("before %d size %d: seqs = %v, want %v", beforeSeq, size, got, want)
		}
	}
}

func assertInboxUnseen(t *testing.T, repo *NotificationInboxRepository, userSeq, want int) {
	t.Helper()
	got, err := repo.CountUnseenNotices(userSeq)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("user %d unseen = %d, want %d", userSeq, got, want)
	}
}

func assertInboxStateRows(t *testing.T, db *sqlx.DB, want int) {
	t.Helper()
	var got int
	if err := db.Get(&got, `SELECT COUNT(*) FROM ALUMNI_NOTIFICATION_INBOX_STATE`); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("inbox state rows = %d, want %d", got, want)
	}
}
