// maintenance_write.go — Fail-closed HTTP write blocking during maintenance.
package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/maintenance"
)

// MaintenanceWriteMiddleware rejects write-like requests while maintenance is active.
func MaintenanceWriteMiddleware(gate *maintenance.Gate) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowsSmoke := isLoopbackRemoteAddr(r.RemoteAddr) &&
				gate.AllowsSmoke(r.URL.Path, r.Header.Get(maintenance.SmokeProofHeader))
			if gate.Active() && isMaintenanceWrite(r) && !allowsSmoke {
				w.Header().Set("Retry-After", "60")
				writeError(w, http.StatusServiceUnavailable, "MAINTENANCE_MODE", "Writes are temporarily unavailable")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isLoopbackRemoteAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isMaintenanceWrite(r *http.Request) bool {
	switch r.Method {
	case http.MethodHead, http.MethodOptions:
		return false
	case http.MethodGet:
		switch r.URL.Path {
		case "/api/auth/kakao/callback", "/pg/easypay/relay":
			return true
		default:
			return hasSingleNumericPathSegment(r.URL.Path, "/api/feed/") ||
				hasSingleNumericPathSegment(r.URL.Path, "/api/disclosure/")
		}
	default:
		return true
	}
}

func hasSingleNumericPathSegment(path, prefix string) bool {
	remainder, ok := strings.CutPrefix(path, prefix)
	if !ok || remainder == "" || strings.Contains(remainder, "/") {
		return false
	}
	for _, character := range remainder {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
