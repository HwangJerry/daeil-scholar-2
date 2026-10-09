package middleware

import (
	"github.com/dflh-saf/backend/internal/model"
	"net/http"
	"testing"
)

func TestDeferredLogoutRemainsReachableForBlockedBuild(t *testing.T) {
	policies := forcingPolicies()
	request := gateRequest("/api/auth/logout/deferred", model.AppPlatformIOS, "199", "18.0")
	request.Method = http.MethodPost
	response, reached := serveGate(policies, request)
	if !reached || response.Code != http.StatusOK {
		t.Fatalf("ended session cannot revoke while update blocked: %d", response.Code)
	}
	if len(policies.calls) != 0 {
		t.Fatal("logout cleanup consulted app update policy")
	}
}
