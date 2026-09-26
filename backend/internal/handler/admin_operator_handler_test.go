package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

func operatorRequest(method, target string, actor int, body string) *http.Request {
	request := httptest.NewRequest(method, "/api/admin/operators/"+target, strings.NewReader(body))
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("usrSeq", target)
	ctx := context.WithValue(request.Context(), chi.RouteCtxKey, routeContext)
	ctx = middleware.SetAuthUser(ctx, &model.AuthUser{USRSeq: actor})
	return request.WithContext(ctx)
}

func TestOperatorEndpointsRejectSelfRoleChange(t *testing.T) {
	handler := NewAdminOperatorHandler(service.NewAdminOperatorService(nil))

	put := httptest.NewRecorder()
	handler.SetRole(put, operatorRequest(http.MethodPut, "7", 7, `{"role":"operator"}`))
	del := httptest.NewRecorder()
	handler.Revoke(del, operatorRequest(http.MethodDelete, "7", 7, ""))

	for name, recorder := range map[string]*httptest.ResponseRecorder{"put": put, "delete": del} {
		if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), `"code":"SELF_ROLE_CHANGE"`) {
			t.Fatalf("%s: status = %d, body = %s", name, recorder.Code, recorder.Body.String())
		}
	}
}

func TestOperatorSetRoleRejectsUnknownRole(t *testing.T) {
	handler := NewAdminOperatorHandler(service.NewAdminOperatorService(nil))
	recorder := httptest.NewRecorder()

	handler.SetRole(recorder, operatorRequest(http.MethodPut, "42", 7, `{"role":"superuser"}`))

	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"code":"INVALID_ROLE"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
