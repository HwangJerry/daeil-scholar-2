package service

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestSubmitAlumniVerificationRequiresCompleteAcademicInformation(t *testing.T) {
	service := NewProfileService(nil)

	err := service.SubmitAlumniVerification(42, model.AlumniVerificationSubmissionRequest{})
	if !errors.Is(err, ErrAcademicInformationRequired) {
		t.Fatalf("error = %v, want ErrAcademicInformationRequired", err)
	}
}

func TestSubmitAlumniVerificationRejectsUnknownDepartment(t *testing.T) {
	service := NewProfileService(nil)

	err := service.SubmitAlumniVerification(42, model.AlumniVerificationSubmissionRequest{
		GraduationYear: 2004,
		Cohort:         "18",
		Department:     "존재하지않는학과",
	})
	if !errors.Is(err, ErrInvalidDepartment) {
		t.Fatalf("error = %v, want ErrInvalidDepartment", err)
	}
}

func TestSubmitAlumniVerificationCreatesPendingApplicationAtomically(t *testing.T) {
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
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = service.SubmitAlumniVerification(42, model.AlumniVerificationSubmissionRequest{
		GraduationYear: 2004,
		Cohort:         " 18 ",
		Department:     " 영어 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSubmitAlumniVerificationResubmitsRejectedApplication(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewProfileService(repository.NewProfileRepository(sqlx.NewDb(db, "sqlmock")))

	mock.ExpectBegin()
	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"STATUS"}).AddRow("rejected"))
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

func TestSubmitAlumniVerificationMovesApprovedAcademicChangeToReapproval(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewProfileService(repository.NewProfileRepository(sqlx.NewDb(db, "sqlmock")))

	mock.ExpectBegin()
	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"STATUS", "APPROVED_GRADUATION_YEAR", "APPROVED_COHORT", "APPROVED_DEPARTMENT",
		}).AddRow("approved", 2003, "17", "독일어"))
	mock.ExpectExec(`UPDATE WEO_MEMBER`).
		WithArgs("18", "영어", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_VERIFICATION`).
		WithArgs("reapproval_pending", 2004, "18", "영어", 42).
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

func TestSubmitAlumniVerificationKeepsUnchangedApprovedReview(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewProfileService(repository.NewProfileRepository(sqlx.NewDb(db, "sqlmock")))

	mock.ExpectBegin()
	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"STATUS", "APPROVED_GRADUATION_YEAR", "APPROVED_COHORT", "APPROVED_DEPARTMENT",
		}).AddRow("approved", 2004, "18", "영어"))
	mock.ExpectExec(`UPDATE WEO_MEMBER`).
		WithArgs("18", "영어", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`SET GRADUATION_YEAR = \?, COHORT = \?, DEPARTMENT = \?, UPDATED_AT = NOW\(\)`).
		WithArgs(2004, "18", "영어", 42).
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
