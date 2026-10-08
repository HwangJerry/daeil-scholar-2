// rate_limit.go — IP-based rate limiting middleware for login brute-force protection
package middleware

import (
	"github.com/dflh-saf/backend/internal/observability"
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	loginMaxAttempts     = 10
	loginAttemptKeyPref  = "login_attempts:"
	loginRateLimitWindow = 15 * time.Minute
)

// LoginRateLimiter returns a middleware that limits login attempts per IP to
// loginMaxAttempts within the cache TTL window. Exceeding the limit returns 429.
func LoginRateLimiter(c *cache.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			key := loginAttemptKeyPref + ip + ":" + r.URL.Path

			c.Add(key, 0, loginRateLimitWindow)
			count, _ := c.IncrementInt(key, 1)
			if count > loginMaxAttempts {
				respondError(w, http.StatusTooManyRequests, "RATE_LIMITED", "잠시 후 다시 시도해주세요")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Use only forwarding headers from the local, trusted Apache proxy.
func clientIP(r *http.Request) string { ip, _ := observability.ClientIP(r); return ip }
