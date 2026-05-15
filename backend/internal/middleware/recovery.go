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
// through the supplied zerolog logger (so the standard log pipeline still
// works), forwards the recovered value + stack trace to the Debug Agent if
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
					Interface("panic", rec).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Bytes("stack", stack).
					Msg("http handler panicked")
				hook.ReportPanic(rec, stack, map[string]interface{}{
					"path":   r.URL.Path,
					"method": r.Method,
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"code":"internal","message":"internal server error"}`))
			}()
			next.ServeHTTP(w, r)
		})
	}
}
