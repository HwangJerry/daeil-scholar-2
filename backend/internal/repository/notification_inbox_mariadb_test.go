package repository

import (
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"github.com/jmoiron/sqlx"
)

// TestNotificationInboxOnMariaDB101 runs the inbox queries against the production
// baseline plus migration 078: the 90-day window, the published filter, keyset
// paging and the last-seen upsert all have to hold on the real 10.1 engine.
func TestNotificationInboxOnMariaDB101(t *testing.T) {
	inputs := append(mariadb.ProdBaseline(t), mariadb.File("../../migrations/078_create_notification_inbox_state.sql"))
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
	if lastSeen, err := repo.GetLastSeenAt(42); err != nil || lastSeen != nil {
		t.Fatalf("never-seen member lastSeen = %v, err = %v", lastSeen, err)
	}
	if err := repo.MarkSeen(42); err != nil {
		t.Fatal(err)
	}
	assertInboxUnseen(t, repo, 42, 0)
	assertInboxUnseen(t, repo, 43, 3)
	lastSeen, err := repo.GetLastSeenAt(42)
	if err != nil || lastSeen == nil || lastSeen.IsZero() {
		t.Fatalf("seen member lastSeen = %v, err = %v", lastSeen, err)
	}

	db.MustExec(`UPDATE ALUMNI_NOTIFICATION_INBOX_STATE SET LAST_SEEN_AT = NOW() - INTERVAL 1 DAY WHERE USR_SEQ = 42`)
	assertInboxUnseen(t, repo, 42, 1)
	// Marking again goes through the ON DUPLICATE KEY branch and clears the dot.
	if err := repo.MarkSeen(42); err != nil {
		t.Fatal(err)
	}
	assertInboxUnseen(t, repo, 42, 0)
	assertInboxStateRows(t, db, 1)
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
