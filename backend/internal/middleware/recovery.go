// recovery.go — Recover handler panics and record sanitized local diagnostics.
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog"
)

// Recoverer logs a fixed panic label, route template and runtime stack locally,
// then responds with a generic 500 JSON error without the recovered value.
func Recoverer(logger zerolog.Logger) func(http.Handler) http.Handler {
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
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"code":"internal","message":"internal server error"}`))
			}()
			next.ServeHTTP(w, r)
		})
	}
}
