package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func TestUpdateProfileSavesApprovedAcademicFieldsWithoutChangingApproval(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := NewProfileRepository(sqlx.NewDb(db, "sqlmock"))
	repository.EnablePhoneClaims()
	jobCategory := 3

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_PHONE[\s\S]*FOR UPDATE`).WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_PHONE"}).AddRow("01012345678"))
	mock.ExpectExec(`UPDATE WEO_MEMBER LEFT JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = WEO_MEMBER.USR_SEQ AND v.STATUS = 'approved'[\s\S]*v.COHORT = COALESCE[\s\S]*v.DEPARTMENT = COALESCE`).
		WithArgs(
			"홍길동", "01012345678", "user@example.com",
			"회사", "소개", "주소", "직무", jobCategory, "Y", "N", "99", "영어", "99", "영어", 42,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repository.UpdateProfile(42, model.ProfileUpdateRequest{
		USRName:        "홍길동",
		USRFN:          "99",
		USRPhone:       "01012345678",
		USREmail:       "user@example.com",
		BizName:        "회사",
		BizDesc:        "소개",
		BizAddr:        "주소",
		Position:       "직무",
		FmDept:         "영어",
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
