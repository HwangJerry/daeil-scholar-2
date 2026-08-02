package service

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TestSubmitAlumniVerificationRetriesConcurrentInitialInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewProfileService(repository.NewProfileRepository(sqlx.NewDb(db, "sqlmock")))

	mock.ExpectBegin()
	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`UPDATE WEO_MEMBER`).
		WithArgs("18", "영어", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
		WithArgs(42, "pending", 2004, "18", "영어").
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate verification row"})
	mock.ExpectRollback()

	mock.ExpectBegin()
	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"STATUS", "APPROVED_GRADUATION_YEAR", "APPROVED_COHORT", "APPROVED_DEPARTMENT",
		}).AddRow("pending", nil, nil, nil))
	mock.ExpectExec(`UPDATE WEO_MEMBER`).
		WithArgs("18", "영어", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_VERIFICATION`).
		WithArgs("pending", 2004, "18", "영어", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = service.SubmitAlumniVerification(42, model.AlumniVerificationSubmissionRequest{
		GraduationYear: 2004,
		Cohort:         "18",
		Department:     "영어",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSubmitAlumniVerificationRetriesConcurrentInitialInsertDeadlock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewProfileService(repository.NewProfileRepository(sqlx.NewDb(db, "sqlmock")))

	mock.ExpectBegin()
	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`UPDATE WEO_MEMBER`).
		WithArgs("18", "영어", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
		WithArgs(42, "pending", 2004, "18", "영어").
		WillReturnError(&mysql.MySQLError{Number: 1213, Message: "deadlock found when trying to get lock"})
	mock.ExpectRollback()

	mock.ExpectBegin()
	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"STATUS", "APPROVED_GRADUATION_YEAR", "APPROVED_COHORT", "APPROVED_DEPARTMENT",
		}).AddRow("pending", nil, nil, nil))
	mock.ExpectExec(`UPDATE WEO_MEMBER`).
		WithArgs("18", "영어", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_VERIFICATION`).
		WithArgs("pending", 2004, "18", "영어", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = service.SubmitAlumniVerification(42, model.AlumniVerificationSubmissionRequest{
		GraduationYear: 2004,
		Cohort:         "18",
		Department:     "영어",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
