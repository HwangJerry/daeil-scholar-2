package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestApprovedAlumniMiddlewareAllowsOnlyApprovedStatus(t *testing.T) {
	tests := []struct {
		status   model.VerificationStatus
		wantCode int
		wantNext bool
	}{
		{status: model.VerificationUnsubmitted, wantCode: http.StatusForbidden},
		{status: model.VerificationPending, wantCode: http.StatusForbidden},
		{status: model.VerificationRejected, wantCode: http.StatusForbidden},
		{status: model.VerificationReapprovalPending, wantCode: http.StatusForbidden},
		{status: model.VerificationApproved, wantCode: http.StatusNoContent, wantNext: true},
	}

	for _, test := range tests {
		t.Run(string(test.status), func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			handler := ApprovedAlumniMiddleware(next)
			request := httptest.NewRequest(http.MethodGet, "/api/alumni", nil)
			request = request.WithContext(SetAuthUser(request.Context(), &model.AuthUser{
				Verification: model.AlumniVerification{Status: test.status},
			}))
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != test.wantCode {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			if called != test.wantNext {
				t.Fatalf("next called = %v", called)
			}
			if !test.wantNext && !strings.Contains(recorder.Body.String(), `"code":"ALUMNI_APPROVAL_REQUIRED"`) {
				t.Fatalf("body = %s", recorder.Body.String())
			}
		})
	}
}
