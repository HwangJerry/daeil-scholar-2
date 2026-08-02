package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
)

func TestGetAlumniVerificationAllowsLimitedSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewProfileHandler(service.NewProfileService(repository.NewProfileRepository(sqlx.NewDb(db, "sqlmock"))))

	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow("pending", 2004, "18", "영어", nil, nil, nil))

	request := httptest.NewRequest(http.MethodGet, "/api/alumni/verification", nil)
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42}))
	recorder := httptest.NewRecorder()

	handler.GetAlumniVerification(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPutAlumniVerificationReturnsPendingObject(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewProfileHandler(service.NewProfileService(repository.NewProfileRepository(sqlx.NewDb(db, "sqlmock"))))

	mock.ExpectBegin()
	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION`).WithArgs(42).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`UPDATE WEO_MEMBER`).WithArgs("18", "영어", 42).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
		WithArgs(42, "pending", 2004, "18", "영어").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow("pending", 2004, "18", "영어", nil, nil, nil))

	request := httptest.NewRequest(http.MethodPut, "/api/alumni/verification", strings.NewReader(`{
		"graduationYear": 2004,
		"cohort": "18",
		"department": "영어"
	}`))
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42}))
	recorder := httptest.NewRecorder()

	handler.PutAlumniVerification(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"status":"pending"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
