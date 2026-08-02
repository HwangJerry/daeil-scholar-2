package repository

import (
	"database/sql"
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

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WillReturnResult(sqlmock.NewResult(42, 1))
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

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WillReturnResult(sqlmock.NewResult(42, 1))
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

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WillReturnResult(sqlmock.NewResult(42, 1))
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

func TestMergeSocialAccountDoesNotWriteAcademicFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	jobCategory := 3

	mock.ExpectBegin()
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "subject").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`SET USR_NAME = \?, USR_EMAIL = \?, USR_JOB_CAT = \?`).
		WithArgs("홍길동", "member@example.com", &jobCategory, "회사", "소개", "주소", "직무", "N", "N", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WillReturnError(errors.New("stop after member update"))
	mock.ExpectRollback()

	_, err = repository.MergeSocialAccount(SocialAccountFields{
		USRSeq:   42,
		Provider: "KT",
		SocialID: "subject",
		Name:     "홍길동",
		Email:    "member@example.com",
		FN:       "99",
		FmDept:   "변경학과",
		JobCat:   &jobCategory,
		BizName:  "회사",
		BizDesc:  "소개",
		BizAddr:  "주소",
		Position: "직무",
	})
	if err == nil {
		t.Fatal("expected social insert failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateMemberMergeFieldsDoesNotWriteAcademicFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	jobCategory := 3

	mock.ExpectExec(`SET USR_NAME = \?, USR_EMAIL = \?, USR_JOB_CAT = \?`).
		WithArgs("홍길동", "member@example.com", &jobCategory, "회사", "소개", "주소", "직무", "N", "N", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repository.UpdateMemberMergeFields(
		42, "홍길동", "member@example.com", "99", "변경학과", &jobCategory,
		"회사", "소개", "주소", "직무", "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
