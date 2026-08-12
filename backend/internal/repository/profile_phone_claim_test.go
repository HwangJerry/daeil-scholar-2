package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TestUpdateProfileTransfersCanonicalPhoneClaimAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewProfileRepository(sqlx.NewDb(db, "sqlmock"))
	repo.EnablePhoneClaims()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_PHONE[\s\S]*FOR UPDATE`).WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_PHONE"}).AddRow("010-1111-2222"))
	mock.ExpectExec(`UPDATE AUTH_PHONE_CLAIM`).WithArgs("01033334444", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE WEO_MEMBER`).WithArgs("Member", "01033334444", "m@example.com", "", "", "", "", 0, "Y", "Y", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repo.UpdateProfile(42, model.ProfileUpdateRequest{USRName: "Member", USRPhone: "010-3333-4444", USREmail: "m@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateProfileRejectsClaimedPhoneAndRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewProfileRepository(sqlx.NewDb(db, "sqlmock"))
	repo.EnablePhoneClaims()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_PHONE[\s\S]*FOR UPDATE`).WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_PHONE"}).AddRow("01011112222"))
	mock.ExpectExec(`UPDATE AUTH_PHONE_CLAIM`).WithArgs("01033334444", 42).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate phone"})
	mock.ExpectRollback()

	err = repo.UpdateProfile(42, model.ProfileUpdateRequest{USRPhone: "01033334444"})
	if !errors.Is(err, ErrPhoneAlreadyClaimed) {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateProfileBeforePhoneClaimCutoverDoesNotRequireNewTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewProfileRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`UPDATE WEO_MEMBER`).
		WithArgs("Member", "01033334444", "m@example.com", "", "", "", "", 0, "Y", "Y", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateProfile(42, model.ProfileUpdateRequest{USRName: "Member", USRPhone: "010-3333-4444", USREmail: "m@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateProfileRejectsInvalidCanonicalPhoneBeforeDatabaseWrite(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewProfileRepository(sqlx.NewDb(db, "sqlmock"))
	repo.EnablePhoneClaims()

	err = repo.UpdateProfile(42, model.ProfileUpdateRequest{USRPhone: "not-a-phone"})
	if !errors.Is(err, ErrInvalidPhone) {
		t.Fatalf("error = %v, want ErrInvalidPhone", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateProfileFailsClosedWhilePhoneClaimMigrationIsStarted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewProfileRepository(sqlx.NewDb(db, "sqlmock"))
	repo.EnablePhoneClaimAutoDetection()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT state FROM _migration_journal[\s\S]*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"state"}).AddRow("STARTED"))
	mock.ExpectRollback()

	err = repo.UpdateProfile(42, model.ProfileUpdateRequest{USRPhone: "01012345678"})
	if !errors.Is(err, ErrPhoneClaimsMigrating) {
		t.Fatalf("error = %v, want ErrPhoneClaimsMigrating", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
