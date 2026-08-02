package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

func TestRejectAlumniVerificationRequiresReason(t *testing.T) {
	handler := NewAdminMemberHandler(service.NewAdminMemberService(nil))
	request := httptest.NewRequest(http.MethodPost, "/api/admin/alumni-verifications/42/reject", strings.NewReader(`{
		"reason": "   ",
		"expectedUpdatedAt": "2026-07-27T01:00:00Z"
	}`))
	request = withAdminVerificationRouteContext(request, "42", 7)
	recorder := httptest.NewRecorder()

	handler.RejectAlumniVerification(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"REJECTION_REASON_REQUIRED"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestRejectAlumniVerificationMapsStaleReview(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewAdminMemberHandler(service.NewAdminMemberService(repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))))
	expectedUpdatedAt := time.Date(2026, time.July, 27, 1, 0, 0, 0, time.UTC)

	mock.ExpectExec(`UPDATE ALUMNI_VERIFICATION`).
		WithArgs("rejected", "사유", 7, 42, expectedUpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT STATUS, UPDATED_AT`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"STATUS", "UPDATED_AT"}).
			AddRow("pending", expectedUpdatedAt.Add(time.Minute)))

	request := httptest.NewRequest(http.MethodPost, "/api/admin/alumni-verifications/42/reject", strings.NewReader(`{
		"reason": "사유",
		"expectedUpdatedAt": "2026-07-27T01:00:00Z"
	}`))
	request = withAdminVerificationRouteContext(request, "42", 7)
	recorder := httptest.NewRecorder()

	handler.RejectAlumniVerification(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"VERIFICATION_STALE"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRejectAlumniVerificationRequiresExpectedUpdatedAt(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewAdminMemberHandler(service.NewAdminMemberService(repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))))
	request := httptest.NewRequest(http.MethodPost, "/api/admin/alumni-verifications/42/reject", strings.NewReader(`{"reason":"사유"}`))
	request = withAdminVerificationRouteContext(request, "42", 7)
	recorder := httptest.NewRecorder()

	handler.RejectAlumniVerification(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"INVALID_BODY"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestApproveAlumniVerificationReturnsNoContent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewAdminMemberHandler(service.NewAdminMemberService(repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))))
	expectedUpdatedAt := time.Date(2026, time.July, 27, 1, 0, 0, 0, time.UTC)

	mock.ExpectExec(`APPROVED_GRADUATION_YEAR = GRADUATION_YEAR`).
		WithArgs("approved", 7, 42, expectedUpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	request := httptest.NewRequest(http.MethodPost, "/api/admin/alumni-verifications/42/approve", strings.NewReader(`{
		"expectedUpdatedAt": "2026-07-27T01:00:00Z"
	}`))
	request = withAdminVerificationRouteContext(request, "42", 7)
	recorder := httptest.NewRecorder()

	handler.ApproveAlumniVerification(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApproveAlumniVerificationMapsStateConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewAdminMemberHandler(service.NewAdminMemberService(repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))))
	expectedUpdatedAt := time.Date(2026, time.July, 27, 1, 0, 0, 0, time.UTC)

	mock.ExpectExec(`APPROVED_GRADUATION_YEAR = GRADUATION_YEAR`).
		WithArgs("approved", 7, 42, expectedUpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT STATUS, UPDATED_AT`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"STATUS", "UPDATED_AT"}).AddRow("approved", expectedUpdatedAt))

	request := httptest.NewRequest(http.MethodPost, "/api/admin/alumni-verifications/42/approve", strings.NewReader(`{
		"expectedUpdatedAt": "2026-07-27T01:00:00Z"
	}`))
	request = withAdminVerificationRouteContext(request, "42", 7)
	recorder := httptest.NewRecorder()

	handler.ApproveAlumniVerification(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"VERIFICATION_STATE_CONFLICT"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApproveAlumniVerificationRequiresExpectedUpdatedAt(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewAdminMemberHandler(service.NewAdminMemberService(repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))))
	request := httptest.NewRequest(http.MethodPost, "/api/admin/alumni-verifications/42/approve", strings.NewReader(`{}`))
	request = withAdminVerificationRouteContext(request, "42", 7)
	recorder := httptest.NewRecorder()

	handler.ApproveAlumniVerification(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"INVALID_BODY"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestListAlumniVerificationsFiltersCanonicalPendingStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewAdminMemberHandler(service.NewAdminMemberService(repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))))

	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION v`).
		WithArgs("pending", "pending").
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_NAME", "STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT",
			"REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT", "UPDATED_AT",
		}).AddRow(42, "홍길동", "pending", 2004, "18", "영어", nil, expectedTimeForList(), nil, expectedTimeForList()))

	request := httptest.NewRequest(http.MethodGet, "/api/admin/alumni-verifications?status=pending", nil)
	recorder := httptest.NewRecorder()

	handler.ListAlumniVerifications(recorder, request)

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

func TestGetAlumniVerificationDetailReturnsUpdatedAt(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewAdminMemberHandler(service.NewAdminMemberService(repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))))
	updatedAt := expectedTimeForList()

	mock.ExpectQuery(`WHERE v.USR_SEQ = \?`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_NAME", "STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT",
			"REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT", "UPDATED_AT",
		}).AddRow(42, "홍길동", "pending", 2004, "18", "영어", nil, updatedAt, nil, updatedAt))

	request := httptest.NewRequest(http.MethodGet, "/api/admin/alumni-verifications/42", nil)
	request = withAdminVerificationRouteContext(request, "42", 7)
	recorder := httptest.NewRecorder()

	handler.GetAlumniVerificationDetail(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"updatedAt":"2026-07-27T01:00:00Z"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyMemberUpdateCannotSetVerificationStatus(t *testing.T) {
	handler := NewAdminMemberHandler(service.NewAdminMemberService(nil))
	request := httptest.NewRequest(http.MethodPut, "/api/admin/member/42", strings.NewReader(`{"status":"CCC"}`))
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("seq", "42")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	recorder := httptest.NewRecorder()

	handler.Update(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"VERIFICATION_STATE_CONFLICT"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func expectedTimeForList() time.Time {
	return time.Date(2026, time.July, 27, 1, 0, 0, 0, time.UTC)
}

func withAdminVerificationRouteContext(request *http.Request, userSeq string, reviewerSeq int) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("userSeq", userSeq)
	ctx := context.WithValue(request.Context(), chi.RouteCtxKey, routeContext)
	ctx = middleware.SetAuthUser(ctx, &model.AuthUser{USRSeq: reviewerSeq})
	return request.WithContext(ctx)
}
