package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func TestCreateSocialAccountRollsBackWhenVerificationCompanionInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	repository.EnablePhoneClaims()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO AUTH_PHONE_CLAIM`).
		WithArgs("01012345678", 42).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
		WithArgs(42, model.VerificationUnsubmitted).
		WillReturnError(errors.New("companion insert failed"))
	mock.ExpectRollback()

	_, err = repository.CreateSocialAccount(SocialAccountFields{
		Provider:    "KT",
		SocialID:    "subject",
		SocialEmail: "social@example.com",
		USRID:       "member@example.com",
		Name:        "홍길동",
		Phone:       "01012345678",
		Email:       "member@example.com",
	})
	if err == nil {
		t.Fatal("expected companion insert failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInsertMemberRollsBackWhenVerificationCompanionInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	repository.EnablePhoneClaims()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO AUTH_PHONE_CLAIM`).
		WithArgs("01012345678", 42).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
		WithArgs(42, model.VerificationUnsubmitted).
		WillReturnError(errors.New("companion insert failed"))
	mock.ExpectRollback()

	_, err = repository.InsertMember(
		"member@example.com", "홍길동", "01012345678", "", "member@example.com", "", nil,
		"", "", "", "", "", "", "",
	)
	if err == nil {
		t.Fatal("expected companion insert failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInsertMemberWithPwdRollsBackWhenVerificationCompanionInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	repository.EnablePhoneClaims()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO AUTH_PHONE_CLAIM`).
		WithArgs("01012345678", 42).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
		WithArgs(42, model.VerificationUnsubmitted).
		WillReturnError(errors.New("companion insert failed"))
	mock.ExpectRollback()

	_, err = repository.InsertMemberWithPwd(model.RegisterRequest{
		UsrID: "member@example.com",
		Name:  "홍길동",
		Phone: "01012345678",
		Email: "member@example.com",
	}, "hashed-password")
	if err == nil {
		t.Fatal("expected companion insert failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInsertMemberWithPwdBeforePhoneClaimCutoverDoesNotRequireNewTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
		WithArgs(42, model.VerificationUnsubmitted).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	usrSeq, err := repository.InsertMemberWithPwd(model.RegisterRequest{
		UsrID: "member", Name: "홍길동", Phone: "01012345678", Email: "member@example.com",
	}, "hashed-password")
	if err != nil || usrSeq != 42 {
		t.Fatalf("usrSeq = %d, err = %v", usrSeq, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInsertMemberWithPwdFailsClosedWhilePhoneClaimMigrationIsStarted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	repository.EnablePhoneClaimAutoDetection()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT state FROM _migration_journal[\s\S]*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"state"}).AddRow("STARTED"))
	mock.ExpectRollback()

	_, err = repository.InsertMemberWithPwd(model.RegisterRequest{Phone: "01012345678"}, "hashed-password")
	if !errors.Is(err, ErrPhoneClaimsMigrating) {
		t.Fatalf("error = %v, want ErrPhoneClaimsMigrating", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
