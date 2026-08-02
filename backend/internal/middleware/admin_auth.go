// AdminAuthMiddleware authorizes root and operator roles independently of member status.
package middleware

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
)

func AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetAuthUser(r.Context())
		if user == nil {
			respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
			return
		}
		if user.AdminRole == nil || (*user.AdminRole != model.AdminRoleRoot && *user.AdminRole != model.AdminRoleOperator) {
			respondError(w, http.StatusForbidden, "ADMIN_ROLE_REQUIRED", "관리자 권한이 필요합니다")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RootOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetAuthUser(r.Context())
		if user == nil {
			respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
			return
		}
		if user.AdminRole == nil || *user.AdminRole != model.AdminRoleRoot {
			respondError(w, http.StatusForbidden, "ADMIN_ROLE_REQUIRED", "root 권한이 필요합니다")
			return
		}
		next.ServeHTTP(w, r)
	})
}
