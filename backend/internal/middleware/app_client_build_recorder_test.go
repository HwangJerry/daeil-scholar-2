// app_client_build_recorder_test.go — Which requests contribute observed builds.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

type gateObserverStub struct {
	platforms []string
	builds    []int64
	versions  []string
}

func (s *gateObserverStub) Observe(platform string, build int64, versionName string) {
	s.platforms = append(s.platforms, platform)
	s.builds = append(s.builds, build)
	s.versions = append(s.versions, versionName)
}

func serveRecorder(observer AppClientBuildObserver, request *http.Request) bool {
	reached := false
	handler := AppClientBuildRecorder(observer)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), request)
	return reached
}

func authenticatedGateRequest(platform, build, version string) *http.Request {
	request := gateRequest("/api/feed", platform, build, "18.0")
	request.Header.Set(HeaderAppVersion, version)
	return request.WithContext(SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 7}))
}

func TestAppClientBuildRecorderRecordsAuthenticatedClients(t *testing.T) {
	observer := &gateObserverStub{}

	if !serveRecorder(observer, authenticatedGateRequest(model.AppPlatformIOS, "202609151230", "1.0.0")) {
		t.Fatal("handler did not run")
	}

	if len(observer.platforms) != 1 || observer.platforms[0] != model.AppPlatformIOS {
		t.Fatalf("observed platforms = %v", observer.platforms)
	}
	if observer.builds[0] != 202609151230 || observer.versions[0] != "1.0.0" {
		t.Fatalf("observed build = %d, version = %q", observer.builds[0], observer.versions[0])
	}
}

// The recorded maximum bounds the thresholds an administrator may set, so an
// anonymous caller must not be able to introduce a build number.
func TestAppClientBuildRecorderIgnoresUnauthenticatedRequests(t *testing.T) {
	observer := &gateObserverStub{}

	if !serveRecorder(observer, gateRequest("/api/feed", model.AppPlatformIOS, "999999999999", "18.0")) {
		t.Fatal("handler did not run")
	}

	if len(observer.platforms) != 0 {
		t.Fatalf("observed platforms = %v, want none", observer.platforms)
	}
}

func TestAppClientBuildRecorderIgnoresMalformedHeaders(t *testing.T) {
	observer := &gateObserverStub{}

	serveRecorder(observer, authenticatedGateRequest("windows", "10", "1.0.0"))
	serveRecorder(observer, authenticatedGateRequest(model.AppPlatformIOS, "not-a-build", "1.0.0"))

	if len(observer.platforms) != 0 {
		t.Fatalf("observed platforms = %v, want none", observer.platforms)
	}
}

func TestAppClientBuildRecorderPassesThroughWithoutObserver(t *testing.T) {
	if !serveRecorder(nil, authenticatedGateRequest(model.AppPlatformIOS, "10", "1.0.0")) {
		t.Fatal("handler did not run without an observer")
	}
}
