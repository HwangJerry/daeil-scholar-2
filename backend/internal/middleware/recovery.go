// recovery.go — panic recovery middleware that logs via zerolog and forwards
// stack traces to the Debug Agent gateway. Replaces chi's silent Recoverer.
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/dflh-saf/backend/internal/observability"
	"github.com/rs/zerolog"
)

// Recoverer returns middleware that intercepts handler panics, logs them
// through the supplied zerolog logger, forwards a constant panic label and
// stack trace (without the recovered value) to the Debug Agent if
// hook is non-nil, and responds with a generic 500 JSON error.
//
// hook may be nil — in that case only the local zerolog log is emitted.
func Recoverer(logger zerolog.Logger, hook *observability.Hook) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				stack := debug.Stack()
				logger.Error().
					Str("panic", "http handler panicked").
					Str("path", logRoute(r)).
					Str("method", r.Method).
					Bytes("stack", stack).
					Msg("http handler panicked")
				hook.ReportBackendPanic(stack)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"code":"internal","message":"internal server error"}`))
			}()
			next.ServeHTTP(w, r)
		})
	}
}
