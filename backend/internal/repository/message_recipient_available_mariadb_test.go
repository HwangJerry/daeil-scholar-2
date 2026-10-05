// message_recipient_available_mariadb_test.go — Conversation-list recipientAvailable on MariaDB 10.1.
package repository

import (
	"testing"

	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
)

// TestConversationRecipientAvailableOnMariaDB101 checks the per-peer
// RECIPIENT_AVAILABLE flag against the production baseline: approved CCC peers
// are available; withdrawn (AAA) and unapproved peers are not.
func TestConversationRecipientAvailableOnMariaDB101(t *testing.T) {
	db := mariadb.Start(t).NewDatabase(t, mariadb.ProdBaseline(t)...).DB
	db.MustExec(`INSERT INTO WEO_MEMBER (USR_SEQ, USR_ID, USR_NAME, USR_PWD, USR_STATUS) VALUES
		(101, 'me', '요청 동문', 'x', 'CCC'),
		(202, 'ok', '승인 동문', 'x', 'CCC'),
		(303, 'gone', '탈퇴 동문', 'x', 'AAA'),
		(404, 'wait', '대기 동문', 'x', 'CCC')`)
	db.MustExec(`INSERT INTO ALUMNI_VERIFICATION (USR_SEQ, STATUS, CREATED_AT, UPDATED_AT) VALUES
		(101, 'approved', UTC_TIMESTAMP(), UTC_TIMESTAMP()),
		(202, 'approved', UTC_TIMESTAMP(), UTC_TIMESTAMP()),
		(303, 'approved', UTC_TIMESTAMP(), UTC_TIMESTAMP()),
		(404, 'pending', UTC_TIMESTAMP(), UTC_TIMESTAMP())`)
	db.MustExec(`INSERT INTO ALUMNI_MESSAGE (AM_SENDER_SEQ, AM_RECVR_SEQ, AM_CONTENT, AM_CLIENT_MESSAGE_ID, REG_DATE) VALUES
		(101, 202, 'a', 'c-202', UTC_TIMESTAMP()),
		(303, 101, 'b', 'c-303', UTC_TIMESTAMP()),
		(101, 404, 'c', 'c-404', UTC_TIMESTAMP())`)
	db.MustExec(`INSERT INTO ALUMNI_MEMBER_BLOCK (BLOCKER_USR_SEQ, BLOCKED_USR_SEQ, CREATED_AT, UPDATED_AT)
		VALUES (101, 202, UTC_TIMESTAMP(), UTC_TIMESTAMP())`)

	items, err := NewMessageRepository(db).GetConversations(101, nil, 0, 10)
	if err != nil {
		t.Fatalf("GetConversations: %v", err)
	}
	want := map[int]bool{202: true, 303: false, 404: false}
	if len(items) != len(want) {
		t.Fatalf("items = %+v, want %d peers", items, len(want))
	}
	for _, item := range items {
		if item.RecipientAvailable != want[item.UserSeq] {
			t.Errorf("peer %d recipientAvailable = %v, want %v", item.UserSeq, item.RecipientAvailable, want[item.UserSeq])
		}
	}
	for _, item := range items {
		if item.UserSeq == 202 && !item.BlockedByMe {
			t.Error("peer 202 blockedByMe = false, want true (flags are independent)")
		}
	}
}
