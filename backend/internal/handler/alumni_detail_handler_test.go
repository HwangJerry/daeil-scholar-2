package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

func TestGetAlumniDetailRejectsInvalidUserSeqWithCanonicalCode(t *testing.T) {
	router := chi.NewRouter()
	router.Get("/api/alumni/{userSeq}", NewAlumniHandler(nil).GetDetail)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/alumni/not-a-number", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	var response model.APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "INVALID_USER_SEQ" {
		t.Fatalf("code = %q, want INVALID_USER_SEQ", response.Code)
	}
}

func TestGetAlumniDetailHidesMissingOrUnapprovedTargetWithCanonicalCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`JOIN ALUMNI_VERIFICATION v[\s\S]*v.STATUS = 'approved'[\s\S]*m.USR_SEQ = \?`).
		WithArgs(101, 202).
		WillReturnError(sql.ErrNoRows)

	alumniService := service.NewAlumniService(repository.NewAlumniRepository(sqlx.NewDb(db, "sqlmock")), nil)
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := middleware.SetAuthUser(r.Context(), &model.AuthUser{USRSeq: 101})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Get("/api/alumni/{userSeq}", NewAlumniHandler(alumniService).GetDetail)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/alumni/202", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	var response model.APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "INVALID_USER_SEQ" {
		t.Fatalf("code = %q, want INVALID_USER_SEQ", response.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetAlumniDetailUsesCanonicalCodeForRepositoryFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`JOIN ALUMNI_VERIFICATION v[\s\S]*v.STATUS = 'approved'[\s\S]*m.USR_SEQ = \?`).
		WithArgs(101, 202).
		WillReturnError(errors.New("database unavailable"))

	alumniService := service.NewAlumniService(repository.NewAlumniRepository(sqlx.NewDb(db, "sqlmock")), nil)
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := middleware.SetAuthUser(r.Context(), &model.AuthUser{USRSeq: 101})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Get("/api/alumni/{userSeq}", NewAlumniHandler(alumniService).GetDetail)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/alumni/202", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	var response model.APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "INVALID_REQUEST" {
		t.Fatalf("code = %q, want INVALID_REQUEST", response.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
