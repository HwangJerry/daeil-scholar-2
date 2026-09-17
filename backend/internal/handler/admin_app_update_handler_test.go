// admin_app_update_handler_test.go — HTTP contract for app update policy administration.
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type appUpdatePolicyServiceStub struct {
	records    []model.AppUpdatePolicyRecord
	entries    []model.AppUpdatePolicyHistoryEntry
	saveErr    error
	listErr    error
	savedInput service.SaveAppUpdatePolicyInput
	saveCalls  int
}

func (s *appUpdatePolicyServiceStub) ListPolicies() ([]model.AppUpdatePolicyRecord, error) {
	return s.records, s.listErr
}

func (s *appUpdatePolicyServiceStub) SavePolicy(input service.SaveAppUpdatePolicyInput) error {
	s.saveCalls++
	s.savedInput = input
	return s.saveErr
}

func (s *appUpdatePolicyServiceStub) ListHistory(string) ([]model.AppUpdatePolicyHistoryEntry, error) {
	return s.entries, s.listErr
}

type appClientBuildServiceStub struct {
	builds   []model.AppClientBuild
	platform string
	err      error
}

func (s *appClientBuildServiceStub) List(platform string) ([]model.AppClientBuild, error) {
	s.platform = platform
	return s.builds, s.err
}

const savePolicyBody = `{"policy":{"forceEnabled":true,"minBuild":202609151230,"recommendEnabled":false,"recommendedBuild":0,"minOsVersion":"17.0","storeUrl":"https://apps.apple.com/app/id1"},"expectedPolicy":{"forceEnabled":false,"minBuild":202609151000,"recommendEnabled":false,"recommendedBuild":0,"minOsVersion":"17.0","storeUrl":"https://apps.apple.com/app/id1"},"updatedAt":"2026-09-16T10:00:00Z","allowUnobservedBuild":true}`

func savePolicyRouter(handler *AdminAppUpdateHandler) chi.Router {
	router := chi.NewRouter()
	router.Put("/api/admin/app-update-policies/{platform}", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(middleware.SetAuthUser(r.Context(), &model.AuthUser{USRSeq: 7}))
		handler.SavePolicy(w, r)
	})
	return router
}

func TestAdminAppUpdateHandlerListsBothPolicies(t *testing.T) {
	policies := &appUpdatePolicyServiceStub{records: []model.AppUpdatePolicyRecord{
		{Platform: model.AppPlatformIOS, Policy: model.AppUpdatePolicy{MinOSVersion: "17.0"}},
		{Platform: model.AppPlatformAndroid, Policy: model.AppUpdatePolicy{MinOSVersion: "26"}},
	}}
	handler := NewAdminAppUpdateHandler(policies, &appClientBuildServiceStub{})
	response := httptest.NewRecorder()

	handler.ListPolicies(response, httptest.NewRequest(http.MethodGet, "/api/admin/app-update-policies", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Policies []model.AppUpdatePolicyRecord `json:"policies"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Policies) != 2 || payload.Policies[0].Platform != model.AppPlatformIOS {
		t.Fatalf("payload = %#v", payload.Policies)
	}
}

func TestAdminAppUpdateHandlerSavesOnePlatformWithOperatorAndPrecondition(t *testing.T) {
	policies := &appUpdatePolicyServiceStub{}
	handler := NewAdminAppUpdateHandler(policies, &appClientBuildServiceStub{})
	request := httptest.NewRequest(http.MethodPut, "/api/admin/app-update-policies/ios", strings.NewReader(savePolicyBody))
	response := httptest.NewRecorder()

	savePolicyRouter(handler).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	saved := policies.savedInput
	if saved.Platform != model.AppPlatformIOS || saved.UpdatedBy != 7 || !saved.AllowUnobservedBuild {
		t.Fatalf("saved platform = %q, operator = %d, allowUnobserved = %t",
			saved.Platform, saved.UpdatedBy, saved.AllowUnobservedBuild)
	}
	if !saved.Policy.ForceEnabled || saved.Policy.MinBuild != 202609151230 {
		t.Fatalf("saved policy = %#v", saved.Policy)
	}
	if !saved.ExpectedUpdatedAt.Equal(time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("saved updatedAt = %v", saved.ExpectedUpdatedAt)
	}
	if saved.ExpectedPolicy == nil || saved.ExpectedPolicy.MinBuild != 202609151000 {
		t.Fatalf("expected policy = %#v", saved.ExpectedPolicy)
	}
}

// updatedAt is the optimistic-concurrency token, so a body without it is refused
// before it can overwrite a concurrent edit.
func TestAdminAppUpdateHandlerRejectsBodyWithoutPolicyOrUpdatedAt(t *testing.T) {
	bodies := []string{
		`{}`,
		`{"policy":{"minOsVersion":"17.0"}}`,
		`{"updatedAt":"2026-09-16T10:00:00Z"}`,
		// A zero timestamp would skip the conflict check in the repository.
		`{"policy":{"minOsVersion":"17.0"},"updatedAt":"0001-01-01T00:00:00Z"}`,
	}
	for _, body := range bodies {
		policies := &appUpdatePolicyServiceStub{}
		handler := NewAdminAppUpdateHandler(policies, &appClientBuildServiceStub{})
		request := httptest.NewRequest(http.MethodPut, "/api/admin/app-update-policies/ios", strings.NewReader(body))
		response := httptest.NewRecorder()

		savePolicyRouter(handler).ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_BODY"`) {
			t.Fatalf("body %s → status = %d, response = %s", body, response.Code, response.Body.String())
		}
		if policies.saveCalls != 0 {
			t.Fatalf("body %s reached the service", body)
		}
	}
}

func TestAdminAppUpdateHandlerReportsFieldValidationErrors(t *testing.T) {
	policies := &appUpdatePolicyServiceStub{saveErr: &service.AppUpdatePolicyValidationError{
		Fields: []service.AppUpdatePolicyFieldError{{Field: "minBuild", Reason: "이 플랫폼에서 확인된 적 없는 빌드입니다"}},
	}}
	handler := NewAdminAppUpdateHandler(policies, &appClientBuildServiceStub{})
	request := httptest.NewRequest(http.MethodPut, "/api/admin/app-update-policies/ios", strings.NewReader(savePolicyBody))
	response := httptest.NewRecorder()

	savePolicyRouter(handler).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"code":"INVALID_POLICY"`) || !strings.Contains(body, `"field":"minBuild"`) {
		t.Fatalf("body = %s", body)
	}
}

func TestAdminAppUpdateHandlerMapsPolicyErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		status   int
		wantCode string
	}{
		{"conflict", service.ErrAppUpdatePolicyConflict, http.StatusConflict, "POLICY_CONFLICT"},
		{"missing", service.ErrAppUpdatePolicyNotFound, http.StatusNotFound, "POLICY_NOT_FOUND"},
		{"platform", service.ErrUnsupportedAppUpdatePlatform, http.StatusBadRequest, "INVALID_PLATFORM"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewAdminAppUpdateHandler(&appUpdatePolicyServiceStub{saveErr: test.err}, &appClientBuildServiceStub{})
			request := httptest.NewRequest(http.MethodPut, "/api/admin/app-update-policies/ios", strings.NewReader(savePolicyBody))
			response := httptest.NewRecorder()

			savePolicyRouter(handler).ServeHTTP(response, request)

			if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestAdminAppUpdateHandlerListsClientBuildsForRequestedPlatform(t *testing.T) {
	builds := &appClientBuildServiceStub{builds: []model.AppClientBuild{
		{Platform: model.AppPlatformAndroid, Build: 260915001, VersionName: "1.0.0"},
	}}
	handler := NewAdminAppUpdateHandler(&appUpdatePolicyServiceStub{}, builds)
	response := httptest.NewRecorder()

	handler.ListClientBuilds(response, httptest.NewRequest(http.MethodGet, "/api/admin/app-client-builds?platform=android", nil))

	if response.Code != http.StatusOK || builds.platform != model.AppPlatformAndroid {
		t.Fatalf("status = %d, platform = %q", response.Code, builds.platform)
	}
	if !strings.Contains(response.Body.String(), `"build":260915001`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestAdminAppUpdateHandlerListHistoryReturnsEntries(t *testing.T) {
	policies := &appUpdatePolicyServiceStub{entries: []model.AppUpdatePolicyHistoryEntry{
		{Platform: model.AppPlatformIOS, BeforeJSON: "{}", AfterJSON: `{"forceEnabled":true}`},
	}}
	handler := NewAdminAppUpdateHandler(policies, &appClientBuildServiceStub{})
	router := chi.NewRouter()
	router.Get("/api/admin/app-update-policies/{platform}/history", handler.ListHistory)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/admin/app-update-policies/ios/history", nil))

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"forceEnabled\":true`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}
