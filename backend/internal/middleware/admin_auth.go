// AdminAuthMiddleware restricts access to the current canonical administrator
// role loaded by AuthMiddleware. Legacy USR_STATUS remains an account lifecycle
// input, but it is not sufficient authorization for an admin route.
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
		if user.AdminRole == nil || (*user.AdminRole != "root" && *user.AdminRole != "operator") {
			respondError(w, http.StatusForbidden, "FORBIDDEN", "관리자 권한이 필요합니다")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RootAdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetAuthUser(r.Context())
		if user == nil {
			respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
			return
		}
		if user.AdminRole == nil || *user.AdminRole != "root" {
			respondError(w, http.StatusForbidden, "ROOT_REQUIRED", "최고 관리자 권한이 필요합니다")
			return
		}
		next.ServeHTTP(w, r)
	})
}
