package middleware

import (
	"net/http"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

func RequestLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chiMiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			status := ww.Status()
			evt := logger.Info()
			if status >= 500 {
				evt = logger.Error()
			}
			evt.
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", status).
				Dur("duration", time.Since(start)).
				Msg("request")
		})
	}
}
