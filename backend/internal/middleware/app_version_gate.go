// app_version_gate.go — Server-side enforcement of the minimum supported app build.
package middleware

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

// appVersionGateExemptPaths stay reachable from a blocked build: liveness, the
// policy payload itself (otherwise a forced client could not learn it was
// released), and sign-out.
var appVersionGateExemptPaths = map[string]struct{}{
	"/api/health":          {},
	"/api/settings/public": {},
	"/api/auth/logout":     {},
	"/api/auth/logout/all": {},
}

type AppUpdatePolicyProvider interface {
	GetPolicy(platform string) (model.AppUpdatePolicy, error)
}

// AppVersionGate rejects requests from builds below the platform's minimum with
// 426, the second line of defense behind the in-app gate.
//
// It fails open everywhere: missing or malformed headers, an unknown platform,
// and any policy lookup error all let the request through. A backend problem
// must never become a total app outage.
func AppVersionGate(policies AppUpdatePolicyProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers, ok := readClientAppHeaders(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			if _, exempt := appVersionGateExemptPaths[r.URL.Path]; exempt {
				next.ServeHTTP(w, r)
				return
			}

			policy, err := policies.GetPolicy(headers.Platform)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if service.EvaluateAppUpdate(policy, headers.Build, headers.OSVersion) != model.AppUpdateDecisionForce {
				next.ServeHTTP(w, r)
				return
			}
			respondError(
				w,
				http.StatusUpgradeRequired,
				"APP_UPDATE_REQUIRED",
				"최신 버전으로 업데이트한 뒤 이용할 수 있습니다.",
			)
		})
	}
}
