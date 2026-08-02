package repository

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func newMemberBlockRepoMock(t *testing.T) (*MemberBlockRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	return NewMemberBlockRepository(sqlx.NewDb(db, "sqlmock")), mock, func() { _ = db.Close() }
}

func TestMemberBlockRepositoryListUsesDirectionalStableOrdering(t *testing.T) {
	repo, mock, closeDB := newMemberBlockRepoMock(t)
	defer closeDB()
	mock.ExpectQuery(`SELECT BLOCKED_USR_SEQ,[\s\S]*FROM ALUMNI_MEMBER_BLOCK[\s\S]*WHERE BLOCKER_USR_SEQ = \?[\s\S]*ORDER BY UPDATED_AT DESC, BLOCKED_USR_SEQ DESC`).
		WithArgs(101).
		WillReturnRows(sqlmock.NewRows([]string{"BLOCKED_USR_SEQ", "UPDATED_AT"}).
			AddRow(203, "2026-07-29T02:00:00Z").
			AddRow(202, "2026-07-29T01:00:00Z"))

	states, err := repo.List(101)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 2 || states[0].UserSeq != 203 || !states[0].BlockedByMe || states[0].UpdatedAt == nil {
		t.Fatalf("states = %#v", states)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMemberBlockRepositoryGetReturnsNullTimestampWhenNotBlocked(t *testing.T) {
	repo, mock, closeDB := newMemberBlockRepoMock(t)
	defer closeDB()
	mock.ExpectQuery(`SELECT DATE_FORMAT\(UPDATED_AT,[\s\S]*WHERE BLOCKER_USR_SEQ = \? AND BLOCKED_USR_SEQ = \?`).
		WithArgs(101, 202).
		WillReturnError(sql.ErrNoRows)

	state, err := repo.Get(101, 202)
	if err != nil {
		t.Fatal(err)
	}
	if state.UserSeq != 202 || state.BlockedByMe || state.UpdatedAt != nil {
		t.Fatalf("state = %#v", state)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMemberBlockRepositoryBlockIsIdempotentWithoutChangingTimestamp(t *testing.T) {
	repo, mock, closeDB := newMemberBlockRepoMock(t)
	defer closeDB()
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT STATUS[\s\S]*FROM ALUMNI_VERIFICATION[\s\S]*WHERE USR_SEQ = \?[\s\S]*FOR UPDATE`).
		WithArgs(202).
		WillReturnRows(sqlmock.NewRows([]string{"STATUS"}).AddRow("approved"))
	insert := regexp.QuoteMeta(`
		INSERT INTO ALUMNI_MEMBER_BLOCK (
			BLOCKER_USR_SEQ, BLOCKED_USR_SEQ, CREATED_AT, UPDATED_AT
		) VALUES (?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE UPDATED_AT = UPDATED_AT
	`)
	mock.ExpectExec(insert).WithArgs(101, 202).WillReturnResult(sqlmock.NewResult(1, 0))
	mock.ExpectQuery(`SELECT DATE_FORMAT\(UPDATED_AT,[\s\S]*WHERE BLOCKER_USR_SEQ = \? AND BLOCKED_USR_SEQ = \?`).
		WithArgs(101, 202).
		WillReturnRows(sqlmock.NewRows([]string{"UPDATED_AT"}).AddRow("2026-07-29T01:00:00Z"))
	mock.ExpectCommit()

	state, err := repo.Block(101, 202)
	if err != nil {
		t.Fatal(err)
	}
	if !state.BlockedByMe || state.UpdatedAt == nil || *state.UpdatedAt != "2026-07-29T01:00:00Z" {
		t.Fatalf("state = %#v", state)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMemberBlockRepositoryRollsBackUnapprovedTargetWithoutInsert(t *testing.T) {
	repo, mock, closeDB := newMemberBlockRepoMock(t)
	defer closeDB()
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT STATUS[\s\S]*FROM ALUMNI_VERIFICATION[\s\S]*WHERE USR_SEQ = \?[\s\S]*FOR UPDATE`).
		WithArgs(202).
		WillReturnRows(sqlmock.NewRows([]string{"STATUS"}).AddRow("pending"))
	mock.ExpectRollback()

	_, err := repo.Block(101, 202)
	if !errors.Is(err, ErrMemberBlockTargetNotApproved) {
		t.Fatalf("error = %v, want target-not-approved", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMemberBlockRepositoryUnblockIsIdempotent(t *testing.T) {
	repo, mock, closeDB := newMemberBlockRepoMock(t)
	defer closeDB()
	mock.ExpectExec(`DELETE FROM ALUMNI_MEMBER_BLOCK[\s\S]*WHERE BLOCKER_USR_SEQ = \? AND BLOCKED_USR_SEQ = \?`).
		WithArgs(101, 202).
		WillReturnResult(sqlmock.NewResult(0, 0))

	state, err := repo.Unblock(101, 202)
	if err != nil {
		t.Fatal(err)
	}
	if state.BlockedByMe || state.UpdatedAt != nil {
		t.Fatalf("state = %#v", state)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMemberBlockRepositoryChecksCanonicalApprovedTarget(t *testing.T) {
	repo, mock, closeDB := newMemberBlockRepoMock(t)
	defer closeDB()
	mock.ExpectQuery(`SELECT EXISTS\([\s\S]*FROM ALUMNI_VERIFICATION[\s\S]*USR_SEQ = \?[\s\S]*STATUS = 'approved'`).
		WithArgs(202).
		WillReturnRows(sqlmock.NewRows([]string{"approved"}).AddRow(true))

	approved, err := repo.IsApprovedAlumni(202)
	if err != nil {
		t.Fatal(err)
	}
	if !approved {
		t.Fatal("approved = false, want true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMemberBlockRepositoryDeletesOnlyExpiredSuppressedMessagesInSmallBatch(t *testing.T) {
	repo, mock, closeDB := newMemberBlockRepoMock(t)
	defer closeDB()
	mock.ExpectExec(`DELETE FROM ALUMNI_MESSAGE[\s\S]*PURGE_AT IS NOT NULL[\s\S]*PURGE_AT <= UTC_TIMESTAMP\(\)[\s\S]*AM_VISIBLE_RECVR = 'N'[\s\S]*AM_SUPPRESSION_REASON = 'recipient_blocked'[\s\S]*ORDER BY PURGE_AT ASC, AM_SEQ ASC[\s\S]*LIMIT \?`).
		WithArgs(100).
		WillReturnResult(sqlmock.NewResult(0, 37))

	deleted, err := repo.DeleteExpiredSuppressedMessages(100)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 37 {
		t.Fatalf("deleted = %d, want 37", deleted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
