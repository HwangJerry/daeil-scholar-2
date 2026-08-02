package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
)

func TestAlumniReadEndpointsUseCanonicalCodeForRepositoryFailure(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		query      string
		handlerFor func(*AlumniHandler) http.HandlerFunc
	}{
		{name: "search", path: "/api/alumni", query: `SELECT COUNT\(\*\)`, handlerFor: func(h *AlumniHandler) http.HandlerFunc { return h.Search }},
		{name: "filters", path: "/api/alumni/filters", query: `SELECT DISTINCT v.GRADUATION_YEAR`, handlerFor: func(h *AlumniHandler) http.HandlerFunc { return h.GetFilters }},
		{name: "widget", path: "/api/alumni/widget", query: `SELECT COUNT\(\*\)`, handlerFor: func(h *AlumniHandler) http.HandlerFunc { return h.GetWidgetPreview }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectQuery(tt.query).WillReturnError(errors.New("database unavailable"))

			alumniService := service.NewAlumniService(
				repository.NewAlumniRepository(sqlx.NewDb(db, "sqlmock")),
				cache.New(time.Minute, time.Minute),
			)
			handler := NewAlumniHandler(alumniService)
			router := chi.NewRouter()
			router.Get(tt.path, tt.handlerFor(handler))

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))

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
		})
	}
}
