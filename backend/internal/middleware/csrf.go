package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
)

func CSRFMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			origin := r.Header.Get("Origin")
			if origin != "" {
				if !isAllowedCSRFOrigin(origin, allowedOrigins) {
					writeError(w, http.StatusForbidden, "CSRF_REJECTED", "Invalid origin")
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			referer := r.Header.Get("Referer")
			if referer != "" {
				matched := false
				for _, allowed := range allowedOrigins {
					if strings.HasPrefix(referer, allowed) {
						matched = true
						break
					}
				}
				if !matched {
					writeError(w, http.StatusForbidden, "CSRF_REJECTED", "Invalid referer")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedCSRFOrigin(origin string, allowed []string) bool {
	for _, a := range allowed {
		if origin == a {
			return true
		}
	}
	return false
}

func writeError(w http.ResponseWriter, status int, code string, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.APIError{Code: code, Message: msg})
}
