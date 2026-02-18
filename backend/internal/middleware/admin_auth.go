// AdminAuthMiddleware restricts access to operators with USR_STATUS='ZZZ'
package middleware

import (
	"net/http"
)

func AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetAuthUser(r.Context())
		if user == nil {
			respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
			return
		}
		if user.USRStatus != "ZZZ" {
			respondError(w, http.StatusForbidden, "FORBIDDEN", "관리자 권한이 필요합니다")
			return
		}
		next.ServeHTTP(w, r)
	})
}
