package middleware

import (
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

func TestAdminAuthMiddlewareRejectsStaleZZZClaimWithoutCurrentRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	svc := service.NewAuthService(repo, nil, &config.Config{JWT: config.JWTConfig{
		Secret: "fixture-secret",
		MaxAge: time.Hour,
	}}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
	token, err := svc.GenerateJWT(&model.AuthUser{USRSeq: 42, USRStatus: "ZZZ"})
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
		WithArgs(42).
		WillReturnRows(principalRows().AddRow(
			42, "operator", "Operator", "operator@example.com", "ZZZ", nil,
			"approved", nil, nil, nil, nil, nil, nil,
		))

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	request.AddCookie(&http.Cookie{Name: jwtCookieName, Value: token})
	recorder := httptest.NewRecorder()

	AuthMiddleware(svc)(AdminAuthMiddleware(next)).ServeHTTP(recorder, request)

	if called {
		t.Fatal("principal without a current admin role reached admin handler")
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminAuthMiddlewareAllowsCurrentCanonicalRoles(t *testing.T) {
	tests := []struct {
		name        string
		status      string
		currentRole string
	}{
		{name: "root", status: "ZZZ", currentRole: "root"},
		{name: "operator", status: "CCC", currentRole: "operator"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
			svc := service.NewAuthService(repo, nil, &config.Config{JWT: config.JWTConfig{
				Secret: "fixture-secret",
				MaxAge: time.Hour,
			}}, cache.New(time.Minute, time.Minute), nil, zerolog.Nop())
			token, err := svc.GenerateJWT(&model.AuthUser{USRSeq: 42, USRStatus: "BBB"})
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectQuery(regexp.QuoteMeta("SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS")).
				WithArgs(42).
				WillReturnRows(principalRows().AddRow(
					42, "operator", "Operator", "operator@example.com", tt.status, tt.currentRole,
					"approved", nil, nil, nil, nil, nil, nil,
				))

			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
			request.AddCookie(&http.Cookie{Name: jwtCookieName, Value: token})
			recorder := httptest.NewRecorder()

			AuthMiddleware(svc)(AdminAuthMiddleware(next)).ServeHTTP(recorder, request)

			if !called || recorder.Code != http.StatusNoContent {
				t.Fatalf("called = %v status = %d body = %s", called, recorder.Code, recorder.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRootAdminAuthMiddlewareRejectsCurrentOperator(t *testing.T) {
	role := model.AdminRole("operator")
	request := httptest.NewRequest(http.MethodPut, "/api/admin/member/42", nil)
	request = request.WithContext(SetAuthUser(request.Context(), &model.AuthUser{
		USRSeq:    7,
		USRStatus: "CCC",
		AdminRole: &role,
	}))
	recorder := httptest.NewRecorder()

	RootAdminAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRootAdminAuthMiddlewareAllowsCurrentRoot(t *testing.T) {
	role := model.AdminRole("root")
	request := httptest.NewRequest(http.MethodPut, "/api/admin/member/42", nil)
	request = request.WithContext(SetAuthUser(request.Context(), &model.AuthUser{
		USRSeq:    7,
		USRStatus: "ZZZ",
		AdminRole: &role,
	}))
	recorder := httptest.NewRecorder()

	RootAdminAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}
