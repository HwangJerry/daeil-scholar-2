package repository

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestAcceptMessage_AtomicallyPersistsRecipientBlockSuppression(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewMessageRepository(sqlx.NewDb(db, "sqlmock"))

	insert := regexp.QuoteMeta(`
		INSERT INTO ALUMNI_MESSAGE (
			AM_SENDER_SEQ, AM_RECVR_SEQ, AM_CLIENT_MESSAGE_ID,
			AM_CONTENT, AM_READ_YN, AM_VISIBLE_RECVR,
			AM_SUPPRESSION_REASON, PURGE_AT, REG_DATE
		)
		SELECT ?, ?, ?, ?, 'N',
			CASE WHEN block_state.blocked = 1 THEN 'N' ELSE 'Y' END,
			CASE WHEN block_state.blocked = 1 THEN 'recipient_blocked' ELSE NULL END,
			CASE WHEN block_state.blocked = 1 THEN DATE_ADD(UTC_TIMESTAMP(), INTERVAL 1 MONTH) ELSE NULL END,
			UTC_TIMESTAMP()
		FROM (
			SELECT EXISTS(
				SELECT 1 FROM ALUMNI_MEMBER_BLOCK
				WHERE BLOCKER_USR_SEQ = ? AND BLOCKED_USR_SEQ = ?
			) AS blocked
		) AS block_state
		ON DUPLICATE KEY UPDATE AM_SEQ = LAST_INSERT_ID(AM_SEQ)
	`)
	mock.ExpectExec(insert).
		WithArgs(101, 202, "client-1", "안녕하세요.", 202, 101).
		WillReturnResult(sqlmock.NewResult(9001, 1))

	selectAccepted := regexp.QuoteMeta(`
		SELECT AM_SEQ, AM_CLIENT_MESSAGE_ID, AM_VISIBLE_RECVR,
			DATE_FORMAT(REG_DATE, '%Y-%m-%dT%H:%i:%sZ') AS CREATED_AT
		FROM ALUMNI_MESSAGE
		WHERE AM_SENDER_SEQ = ? AND AM_CLIENT_MESSAGE_ID = ?
		LIMIT 1
	`)
	mock.ExpectQuery(selectAccepted).
		WithArgs(101, "client-1").
		WillReturnRows(sqlmock.NewRows([]string{"AM_SEQ", "AM_CLIENT_MESSAGE_ID", "AM_VISIBLE_RECVR", "CREATED_AT"}).
			AddRow(9001, "client-1", "N", "2026-07-28T01:00:00Z"))

	accepted, err := repo.AcceptMessage(101, 202, "client-1", "안녕하세요.")
	if err != nil {
		t.Fatalf("AcceptMessage: %v", err)
	}
	if accepted == nil || accepted.MessageID != 9001 {
		t.Fatalf("accepted = %+v", accepted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL contract mismatch: %v", err)
	}
}

func TestGetInbox_ExcludesSuppressedRecipientMessages(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	repo := NewMessageRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM ALUMNI_MESSAGE
		WHERE AM_RECVR_SEQ = ? AND AM_DEL_RECVR = 'N' AND AM_VISIBLE_RECVR = 'Y'
	`)).WithArgs(202).WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT am.AM_SEQ, am.AM_SENDER_SEQ, am.AM_RECVR_SEQ, am.AM_CONTENT,
			am.AM_READ_YN,
			IFNULL(DATE_FORMAT(am.REG_DATE, '%Y-%m-%d %H:%i:%s'), '') AS REG_DATE,
			IFNULL(DATE_FORMAT(am.READ_DATE, '%Y-%m-%d %H:%i:%s'), '') AS READ_DATE,
			IFNULL(s.USR_NAME, '') AS SENDER_NAME,
			IFNULL(rv.USR_NAME, '') AS RECVR_NAME
		FROM ALUMNI_MESSAGE am
		LEFT JOIN WEO_MEMBER s ON am.AM_SENDER_SEQ = s.USR_SEQ
		LEFT JOIN WEO_MEMBER rv ON am.AM_RECVR_SEQ = rv.USR_SEQ
		WHERE am.AM_RECVR_SEQ = ? AND am.AM_DEL_RECVR = 'N' AND am.AM_VISIBLE_RECVR = 'Y'
		ORDER BY am.REG_DATE DESC
		LIMIT ? OFFSET ?
	`)).WithArgs(202, 20, 0).WillReturnRows(sqlmock.NewRows([]string{
		"AM_SEQ", "AM_SENDER_SEQ", "AM_RECVR_SEQ", "AM_CONTENT", "AM_READ_YN",
		"REG_DATE", "READ_DATE", "SENDER_NAME", "RECVR_NAME",
	}))

	if _, _, err := repo.GetInbox(202, 1, 20); err != nil {
		t.Fatalf("GetInbox: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL contract mismatch: %v", err)
	}
}

func TestGetConversationMessages_KeepsSuppressedOutgoingAndExcludesSuppressedIncoming(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	repo := NewMessageRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM ALUMNI_MESSAGE
		WHERE ((AM_SENDER_SEQ = ? AND AM_RECVR_SEQ = ? AND AM_DEL_SENDER = 'N')
			OR (AM_SENDER_SEQ = ? AND AM_RECVR_SEQ = ? AND AM_DEL_RECVR = 'N' AND AM_VISIBLE_RECVR = 'Y'))
	`)).WithArgs(101, 202, 202, 101).WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT m.AM_SEQ, m.AM_SENDER_SEQ, m.AM_RECVR_SEQ, m.AM_CONTENT, m.AM_READ_YN,
			IFNULL(DATE_FORMAT(m.REG_DATE, '%Y-%m-%d %H:%i:%s'), '') AS REG_DATE,
			IFNULL(DATE_FORMAT(m.READ_DATE, '%Y-%m-%d %H:%i:%s'), '') AS READ_DATE,
			IFNULL(s.USR_NAME, '') AS SENDER_NAME,
			IFNULL(r2.USR_NAME, '') AS RECVR_NAME
		FROM ALUMNI_MESSAGE m
		LEFT JOIN WEO_MEMBER s ON s.USR_SEQ = m.AM_SENDER_SEQ
		LEFT JOIN WEO_MEMBER r2 ON r2.USR_SEQ = m.AM_RECVR_SEQ
		WHERE ((m.AM_SENDER_SEQ = ? AND m.AM_RECVR_SEQ = ? AND m.AM_DEL_SENDER = 'N')
			OR (m.AM_SENDER_SEQ = ? AND m.AM_RECVR_SEQ = ? AND m.AM_DEL_RECVR = 'N' AND m.AM_VISIBLE_RECVR = 'Y'))
		ORDER BY m.AM_SEQ ASC
		LIMIT ? OFFSET ?
	`)).WithArgs(101, 202, 202, 101, 30, 0).WillReturnRows(sqlmock.NewRows([]string{
		"AM_SEQ", "AM_SENDER_SEQ", "AM_RECVR_SEQ", "AM_CONTENT", "AM_READ_YN",
		"REG_DATE", "READ_DATE", "SENDER_NAME", "RECVR_NAME",
	}))

	if _, _, err := repo.GetConversationMessages(101, 202, 1, 30); err != nil {
		t.Fatalf("GetConversationMessages: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL contract mismatch: %v", err)
	}
}

func TestGetConversations_ReturnsCanonicalStableSummaries(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	query := `(?s)SELECT\s+sub\.other_seq AS USER_SEQ,\s+COALESCE\(NULLIF\(CONVERT\(w\.USR_NAME USING utf8mb4\), _utf8mb4''\), _utf8mb4'탈퇴한 회원'\) AS NAME,.*AS LAST_MESSAGE,.*AS LAST_MESSAGE_AT,.*AS UNREAD_COUNT,.*ALUMNI_MEMBER_BLOCK.*BLOCKER_USR_SEQ.*BLOCKED_USR_SEQ.*AS BLOCKED_BY_ME,.*AS CURSOR_CREATED_AT,.*AS CURSOR_MESSAGE_ID.*SELECT latest\.AM_SEQ.*ORDER BY latest\.REG_DATE DESC, latest\.AM_SEQ DESC\s+LIMIT 1.*LEFT JOIN WEO_MEMBER.*ORDER BY m\.REG_DATE DESC, m\.AM_SEQ DESC\s+LIMIT \?`
	createdAt := time.Date(2026, 7, 28, 1, 0, 0, 0, time.UTC)
	mock.ExpectQuery(query).
		WithArgs(101, 101, 101, 101, 101, 101, 101, 21).
		WillReturnRows(sqlmock.NewRows([]string{
			"USER_SEQ", "NAME", "LAST_MESSAGE", "LAST_MESSAGE_AT", "UNREAD_COUNT", "BLOCKED_BY_ME", "CURSOR_CREATED_AT", "CURSOR_MESSAGE_ID",
		}).AddRow(202, "탈퇴한 회원", "hello", "2026-07-28T01:00:00Z", 0, false, createdAt, int64(9001)))

	repo := NewMessageRepository(sqlx.NewDb(db, "sqlmock"))
	items, err := repo.GetConversations(101, nil, 0, 21)
	if err != nil {
		t.Fatalf("GetConversations: %v", err)
	}
	if len(items) != 1 || items[0].UserSeq != 202 || items[0].Name != "탈퇴한 회원" || items[0].CursorLastMessageID != 9001 {
		t.Fatalf("items = %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMarkConversationRead_RequiresConversationParticipantBoundary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE ALUMNI_MESSAGE AS m
		JOIN ALUMNI_MESSAGE AS boundary ON boundary.AM_SEQ = ?
			AND ((boundary.AM_SENDER_SEQ = ? AND boundary.AM_RECVR_SEQ = ?
				AND boundary.AM_DEL_RECVR = 'N' AND boundary.AM_VISIBLE_RECVR = 'Y')
				OR (boundary.AM_SENDER_SEQ = ? AND boundary.AM_RECVR_SEQ = ?
					AND boundary.AM_DEL_SENDER = 'N'))
		SET m.AM_READ_YN = 'Y', m.READ_DATE = NOW()
		WHERE m.AM_RECVR_SEQ = ? AND m.AM_SENDER_SEQ = ?
			AND m.AM_SEQ <= boundary.AM_SEQ AND m.AM_READ_YN = 'N'
			AND m.AM_DEL_RECVR = 'N' AND m.AM_VISIBLE_RECVR = 'Y'
	`)).WithArgs(int64(9001), 202, 101, 101, 202, 101, 202).WillReturnResult(sqlmock.NewResult(0, 2))

	repo := NewMessageRepository(sqlx.NewDb(db, "sqlmock"))
	changed, err := repo.MarkConversationRead(101, 202, 9001)
	if err != nil {
		t.Fatalf("MarkConversationRead: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetCanonicalConversationMessages_DerivesLegacyClientMessageID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	query := `(?s)SELECT\s+m\.AM_SEQ,\s+COALESCE\(m\.AM_CLIENT_MESSAGE_ID, CONCAT\('legacy-', m\.AM_SEQ\)\) AS AM_CLIENT_MESSAGE_ID,.*ORDER BY m\.REG_DATE DESC, m\.AM_SEQ DESC\s+LIMIT \?`
	mock.ExpectQuery(query).
		WithArgs(101, 202, 202, 101, 31).
		WillReturnRows(sqlmock.NewRows([]string{
			"AM_SEQ", "AM_CLIENT_MESSAGE_ID", "AM_SENDER_SEQ", "AM_RECVR_SEQ", "AM_CONTENT",
			"AM_READ_YN", "REG_DATE", "READ_DATE", "SENDER_NAME", "RECVR_NAME",
		}).AddRow(9001, "legacy-9001", 101, 202, "기존 메시지", "Y",
			"2026-07-28T01:00:00Z", "2026-07-28T01:01:00Z", "보낸 동문", "받은 동문"))

	repo := NewMessageRepository(sqlx.NewDb(db, "sqlmock"))
	items, err := repo.GetCanonicalConversationMessages(101, 202, nil, 0, 31)
	if err != nil {
		t.Fatalf("GetCanonicalConversationMessages: %v", err)
	}
	if len(items) != 1 || items[0].ClientMessageID != "legacy-9001" {
		t.Fatalf("items = %+v, want legacy-9001 clientMessageId", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetCanonicalConversationMessages_ContinuesBeforeStableCursor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	query := `(?s)WHERE \(\(m\.AM_SENDER_SEQ = \?.*AM_VISIBLE_RECVR = 'Y'\)\)\s+AND \(m\.REG_DATE < \? OR \(m\.REG_DATE = \? AND m\.AM_SEQ < \?\)\)\s+ORDER BY m\.REG_DATE DESC, m\.AM_SEQ DESC\s+LIMIT \?`
	before := time.Date(2026, 7, 28, 1, 0, 0, 0, time.UTC)
	mock.ExpectQuery(query).
		WithArgs(101, 202, 202, 101, before, before, int64(9003), 2).
		WillReturnRows(sqlmock.NewRows([]string{
			"AM_SEQ", "AM_CLIENT_MESSAGE_ID", "AM_SENDER_SEQ", "AM_RECVR_SEQ", "AM_CONTENT",
			"AM_READ_YN", "REG_DATE", "READ_DATE", "SENDER_NAME", "RECVR_NAME",
		}).AddRow(9002, "client-9002", 202, 101, "다음 메시지", "N",
			"2026-07-28T01:00:00Z", "", "상대 동문", "요청 동문"))

	repo := NewMessageRepository(sqlx.NewDb(db, "sqlmock"))
	items, err := repo.GetCanonicalConversationMessages(101, 202, &before, 9003, 2)
	if err != nil {
		t.Fatalf("GetCanonicalConversationMessages: %v", err)
	}
	if len(items) != 1 || items[0].AMSeq != 9002 {
		t.Fatalf("items = %+v, want message 9002", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetConversations_ContinuesBeforeStableCursor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	query := `(?s)LEFT JOIN WEO_MEMBER w ON w\.USR_SEQ = sub\.other_seq\s+WHERE \(m\.REG_DATE < \? OR \(m\.REG_DATE = \? AND m\.AM_SEQ < \?\)\)\s+ORDER BY m\.REG_DATE DESC, m\.AM_SEQ DESC\s+LIMIT \?`
	before := time.Date(2026, 7, 28, 1, 0, 0, 0, time.UTC)
	mock.ExpectQuery(query).
		WithArgs(101, 101, 101, 101, 101, 101, 101, before, before, int64(9002), 2).
		WillReturnRows(sqlmock.NewRows([]string{
			"USER_SEQ", "NAME", "LAST_MESSAGE", "LAST_MESSAGE_AT", "UNREAD_COUNT", "BLOCKED_BY_ME", "CURSOR_CREATED_AT", "CURSOR_MESSAGE_ID",
		}).AddRow(303, "다음 동문", "이전 대화", "2026-07-28T00:00:00Z", 0, false, before.Add(-time.Hour), int64(9001)))

	repo := NewMessageRepository(sqlx.NewDb(db, "sqlmock"))
	items, err := repo.GetConversations(101, &before, 9002, 2)
	if err != nil {
		t.Fatalf("GetConversations: %v", err)
	}
	if len(items) != 1 || items[0].CursorLastMessageID != 9001 {
		t.Fatalf("items = %+v, want cursor message 9001", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
