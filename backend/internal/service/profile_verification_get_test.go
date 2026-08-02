package service

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestGetAlumniVerificationReturnsUnsubmittedWhenCompanionRowIsMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewProfileService(repository.NewProfileRepository(sqlx.NewDb(db, "sqlmock")))

	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnError(sql.ErrNoRows)

	verification, err := service.GetAlumniVerification(42)
	if err != nil {
		t.Fatal(err)
	}
	if verification.Status != model.VerificationUnsubmitted {
		t.Fatalf("status = %q", verification.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
