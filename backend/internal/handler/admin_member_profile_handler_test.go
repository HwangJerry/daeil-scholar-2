package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func putMemberProfile(t *testing.T, handler *AdminMemberHandler, seq, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPut, "/api/admin/member/"+seq+"/profile", strings.NewReader(body))
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("seq", seq)
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	recorder := httptest.NewRecorder()
	handler.UpdateProfile(recorder, request)
	return recorder
}

func TestUpdateMemberProfileRejectsBlankNameBeforeDatabase(t *testing.T) {
	handler := NewAdminMemberHandler(service.NewAdminMemberService(nil))

	recorder := putMemberProfile(t, handler, "42", `{"usrName":"  "}`)

	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"code":"INVALID_NAME"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateMemberProfileMapsClaimedPhoneToConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))
	handler := NewAdminMemberHandler(service.NewAdminMemberService(repo))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_PHONE FROM WEO_MEMBER`).WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_PHONE"}).AddRow("01011112222"))
	mock.ExpectQuery(`SELECT state FROM _migration_journal`).
		WillReturnRows(sqlmock.NewRows([]string{"state"}).AddRow("APPLIED"))
	mock.ExpectExec(`UPDATE AUTH_PHONE_CLAIM`).WithArgs("01033334444", 42).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate phone"})
	mock.ExpectRollback()

	recorder := putMemberProfile(t, handler, "42", `{"usrName":"홍길동","usrPhone":"010-3333-4444"}`)

	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), `"code":"PHONE_TAKEN"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
