// log_route.go — Log registered route templates instead of personal URL paths.
package middleware

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func logRoute(r *http.Request) string {
	if route := chi.RouteContext(r.Context()); route != nil {
		if pattern := route.RoutePattern(); pattern != "" && pattern != "/*" {
			return pattern
		}
	}
	return "unmatched"
}
