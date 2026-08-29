package repository

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestSyncAdminRoleForStatusChangeUpsertsOnPromotion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO ALUMNI_ADMIN_ROLE[\s\S]*ON DUPLICATE KEY UPDATE`).
		WithArgs(42, rootAdminRole, 42, 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := sqlx.NewDb(db, "sqlmock").Beginx()
	if err != nil {
		t.Fatal(err)
	}
	if err := syncAdminRoleForStatusChangeTx(tx, 42, sql.NullString{String: "CCC", Valid: true}, "ZZZ"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSyncAdminRoleForStatusChangeDeletesOnDemotion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM ALUMNI_ADMIN_ROLE[\s\S]*ADMIN_ROLE = \?`).
		WithArgs(42, rootAdminRole).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := sqlx.NewDb(db, "sqlmock").Beginx()
	if err != nil {
		t.Fatal(err)
	}
	if err := syncAdminRoleForStatusChangeTx(tx, 42, sql.NullString{String: "ZZZ", Valid: true}, "CCC"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSyncAdminRoleForStatusChangeDoesNothingWhenStatusIsUnchanged(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	tx, err := sqlx.NewDb(db, "sqlmock").Beginx()
	if err != nil {
		t.Fatal(err)
	}
	if err := syncAdminRoleForStatusChangeTx(tx, 42, sql.NullString{String: "ZZZ", Valid: true}, "ZZZ"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertRootAdminMemberCreatesRoleWithMemberAsAuditActor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ, USR_STATUS[\s\S]*FOR UPDATE`).
		WithArgs("admin").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WithArgs("admin", "hash", "관리자").
		WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_ADMIN_ROLE[\s\S]*ON DUPLICATE KEY UPDATE`).
		WithArgs(42, rootAdminRole, 42, 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	usrSeq, err := repo.UpsertRootAdminMember("admin", "hash", "관리자")
	if err != nil {
		t.Fatal(err)
	}
	if usrSeq != 42 {
		t.Fatalf("usrSeq = %d, want 42", usrSeq)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertRootAdminMemberPromotesExistingMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ, USR_STATUS[\s\S]*FOR UPDATE`).
		WithArgs("admin").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ", "USR_STATUS"}).AddRow(42, "CCC"))
	mock.ExpectExec(`UPDATE WEO_MEMBER[\s\S]*USR_STATUS = 'ZZZ'`).
		WithArgs("new-hash", "관리자", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_ADMIN_ROLE[\s\S]*ON DUPLICATE KEY UPDATE`).
		WithArgs(42, rootAdminRole, 42, 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	usrSeq, err := repo.UpsertRootAdminMember("admin", "new-hash", "관리자")
	if err != nil {
		t.Fatal(err)
	}
	if usrSeq != 42 {
		t.Fatalf("usrSeq = %d, want 42", usrSeq)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
