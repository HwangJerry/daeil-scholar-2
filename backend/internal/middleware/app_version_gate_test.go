// app_version_gate_test.go — Server-side minimum build enforcement.
package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

type gatePolicyStub struct {
	policies map[string]model.AppUpdatePolicy
	err      error
	calls    []string
}

func (s *gatePolicyStub) GetPolicy(platform string) (model.AppUpdatePolicy, error) {
	s.calls = append(s.calls, platform)
	if s.err != nil {
		return model.AppUpdatePolicy{}, s.err
	}
	return s.policies[platform], nil
}

func forcingPolicies() *gatePolicyStub {
	return &gatePolicyStub{policies: map[string]model.AppUpdatePolicy{
		model.AppPlatformIOS: {
			ForceEnabled: true, MinBuild: 200, MinOSVersion: "17.0",
			StoreURL: "https://apps.apple.com/app/id1",
		},
		model.AppPlatformAndroid: {MinOSVersion: "26"},
	}}
}

func gateRequest(path, platform, build, osVersion string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if platform != "" {
		request.Header.Set(HeaderAppPlatform, platform)
	}
	if build != "" {
		request.Header.Set(HeaderAppBuild, build)
	}
	if osVersion != "" {
		request.Header.Set(HeaderAppOSVersion, osVersion)
	}
	return request
}

func serveGate(policies AppUpdatePolicyProvider, request *http.Request) (*httptest.ResponseRecorder, bool) {
	reached := false
	handler := AppVersionGate(policies)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response, reached
}

func TestAppVersionGateBlocksBuildBelowMinimum(t *testing.T) {
	response, reached := serveGate(forcingPolicies(), gateRequest("/api/feed", model.AppPlatformIOS, "199", "18.0"))

	if reached {
		t.Fatal("handler ran for a blocked build")
	}
	if response.Code != http.StatusUpgradeRequired {
		t.Fatalf("status = %d, want 426", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"code":"APP_UPDATE_REQUIRED"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

// The web SPA sends no client headers and must never be evaluated.
func TestAppVersionGateIgnoresRequestsWithoutClientHeaders(t *testing.T) {
	policies := forcingPolicies()

	_, reached := serveGate(policies, gateRequest("/api/feed", "", "", ""))

	if !reached {
		t.Fatal("handler did not run for a browser request")
	}
	if len(policies.calls) != 0 {
		t.Fatalf("policy lookups = %v, want none", policies.calls)
	}
}

func TestAppVersionGatePassesMalformedHeaders(t *testing.T) {
	tests := []struct{ name, platform, build string }{
		{"unknown platform", "windows", "199"},
		{"non numeric build", model.AppPlatformIOS, "1.0.0"},
		{"zero build", model.AppPlatformIOS, "0"},
		{"negative build", model.AppPlatformIOS, "-3"},
		{"missing build", model.AppPlatformIOS, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, reached := serveGate(forcingPolicies(), gateRequest("/api/feed", test.platform, test.build, "18.0"))
			if !reached {
				t.Fatal("handler did not run")
			}
		})
	}
}

func TestAppVersionGateKeepsExemptPathsReachable(t *testing.T) {
	for path := range appVersionGateExemptPaths {
		t.Run(path, func(t *testing.T) {
			_, reached := serveGate(forcingPolicies(), gateRequest(path, model.AppPlatformIOS, "199", "18.0"))
			if !reached {
				t.Fatalf("%s was blocked", path)
			}
		})
	}
}

// A policy lookup failure must not take the app away from every user.
func TestAppVersionGateFailsOpenOnPolicyError(t *testing.T) {
	policies := forcingPolicies()
	policies.err = errors.New("settings unavailable")

	_, reached := serveGate(policies, gateRequest("/api/feed", model.AppPlatformIOS, "199", "18.0"))

	if !reached {
		t.Fatal("handler did not run when the policy lookup failed")
	}
}

func TestAppVersionGateLeavesOtherPlatformUntouched(t *testing.T) {
	_, reached := serveGate(forcingPolicies(), gateRequest("/api/feed", model.AppPlatformAndroid, "1", "26"))

	if !reached {
		t.Fatal("android request was blocked by the iOS policy")
	}
}

func TestAppVersionGateDoesNotBlockUnsupportedOSDevices(t *testing.T) {
	_, reached := serveGate(forcingPolicies(), gateRequest("/api/feed", model.AppPlatformIOS, "199", "16.7"))

	if !reached {
		t.Fatal("a device that cannot update was blocked")
	}
}

func TestAppVersionGateNormalizesPlatformHeader(t *testing.T) {
	response, reached := serveGate(forcingPolicies(), gateRequest("/api/feed", " IOS ", "199", "18.0"))

	if reached || response.Code != http.StatusUpgradeRequired {
		t.Fatalf("status = %d, handler reached = %t", response.Code, reached)
	}
}
