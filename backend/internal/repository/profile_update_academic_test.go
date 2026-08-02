package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func TestUpdateProfileDoesNotWriteAcademicFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := NewProfileRepository(sqlx.NewDb(db, "sqlmock"))
	jobCategory := 3

	mock.ExpectExec(`SET USR_NAME = \?, USR_PHONE = \?, USR_EMAIL = \?,`).
		WithArgs(
			"홍길동", "01012345678", "user@example.com",
			"회사", "소개", "주소", "직무", jobCategory, "Y", "N", 42,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repository.UpdateProfile(42, model.ProfileUpdateRequest{
		USRName:        "홍길동",
		USRFN:          "99",
		USRPhone:       "01012345678",
		USREmail:       "user@example.com",
		BizName:        "회사",
		BizDesc:        "소개",
		BizAddr:        "주소",
		Position:       "직무",
		FmDept:         "변경학과",
		JobCat:         &jobCategory,
		USRPhonePublic: "Y",
		USREmailPublic: "N",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
