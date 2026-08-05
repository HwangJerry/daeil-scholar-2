package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

func TestAlumniApprovedMiddlewareRejectsPendingPrincipal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	svc := service.NewAuthService(repo, nil, &config.Config{}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())

	mock.ExpectQuery(`SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE\(m.USR_EMAIL, ''\) AS USR_EMAIL, m.USR_STATUS`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "BBB", nil,
			"pending", nil, nil, nil, nil, time.Now(), nil))

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodGet, "/api/messages/inbox", nil)
	request = request.WithContext(SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42}))
	recorder := httptest.NewRecorder()

	AlumniApprovedMiddleware(svc)(next).ServeHTTP(recorder, request)

	if called {
		t.Fatal("pending principal reached approved handler")
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var response model.APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "ALUMNI_APPROVAL_REQUIRED" {
		t.Fatalf("code = %q", response.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlumniApprovedMiddlewareRejectsApprovedHistoryForIneligibleAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	svc := service.NewAuthService(repo, nil, &config.Config{}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_EMAIL", "USR_STATUS", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(42, "member", "Member", "member@example.com", "AAA", nil,
			"approved", nil, nil, nil, nil, nil, nil))

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	request = request.WithContext(SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42, USRStatus: "CCC"}))
	recorder := httptest.NewRecorder()

	AlumniApprovedMiddleware(svc)(next).ServeHTTP(recorder, request)

	if called {
		t.Fatal("approved verification history bypassed current account eligibility")
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
