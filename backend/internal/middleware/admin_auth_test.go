// admin_auth_test.go — Verifies administrator capability checks use canonical roles.
package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestAdminAuthMiddlewareUsesCanonicalRole(t *testing.T) {
	tests := []struct {
		name       string
		role       *model.AdminRole
		usrStatus  string
		wantStatus int
	}{
		{name: "root", role: adminRole(model.AdminRoleRoot), usrStatus: "CCC", wantStatus: http.StatusNoContent},
		{name: "operator", role: adminRole(model.AdminRoleOperator), usrStatus: "CCC", wantStatus: http.StatusNoContent},
		{name: "legacy zzz without role", usrStatus: "ZZZ", wantStatus: http.StatusForbidden},
		{name: "ordinary member", usrStatus: "CCC", wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := AdminAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			request := requestWithAdminUser(&model.AuthUser{AdminRole: test.role, USRStatus: test.usrStatus})
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantStatus == http.StatusForbidden {
				assertAdminRoleRequiredError(t, recorder)
			}
		})
	}
}

func TestRootOnlyMiddlewareRejectsOperator(t *testing.T) {
	tests := []struct {
		name       string
		role       *model.AdminRole
		wantStatus int
	}{
		{name: "root", role: adminRole(model.AdminRoleRoot), wantStatus: http.StatusNoContent},
		{name: "operator", role: adminRole(model.AdminRoleOperator), wantStatus: http.StatusForbidden},
		{name: "no role", wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := RootOnlyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			request := requestWithAdminUser(&model.AuthUser{AdminRole: test.role})
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantStatus == http.StatusForbidden {
				assertAdminRoleRequiredError(t, recorder)
			}
		})
	}
}

func requestWithAdminUser(user *model.AuthUser) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	return request.WithContext(SetAuthUser(request.Context(), user))
}

func adminRole(role model.AdminRole) *model.AdminRole {
	return &role
}

func assertAdminRoleRequiredError(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	var response model.APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "ADMIN_ROLE_REQUIRED" {
		t.Fatalf("error code = %q", response.Code)
	}
}
