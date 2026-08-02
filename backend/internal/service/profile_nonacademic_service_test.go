package service

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestUpdateProfileIgnoresLegacyAcademicFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewProfileService(repository.NewProfileRepository(sqlx.NewDb(db, "sqlmock")))

	mock.ExpectExec(`UPDATE WEO_MEMBER`).
		WithArgs("홍길동", "01012345678", "user@example.com", "", "", "", "직무", 0, "Y", "Y", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = service.UpdateProfile(42, model.ProfileUpdateRequest{
		USRName:  "홍길동",
		USRPhone: "01012345678",
		USREmail: "user@example.com",
		Position: "직무",
		USRFN:    "99",
		FmDept:   "계약에 없는 레거시 학과",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
