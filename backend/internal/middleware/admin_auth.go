// AdminAuthMiddleware restricts access to operators with USR_STATUS='ZZZ'.
//
// Status code reference (authoritative — see admin_member_service.go allowedMemberStatuses):
//   'AAA' = 탈퇴회원 (withdrawn/resigned member)
//   'BBB' = 승인대기 (pending approval)
//   'CCC' = 승인회원 (approved member)
//   'ZZZ' = 운영자 (operator/admin)  ← correct admin status
//
// NOTE: TECH_DESIGN_DOC.md §15.2 incorrectly states the admin status is 'AAA'.
// That is a documentation typo. 'AAA' is the withdrawn-member status; using it
// here would grant admin access to deactivated accounts. The production DB and
// all repository queries (auth_repo.go lines 61, 78, 138) consistently treat
// 'ZZZ' as the operator status. Do NOT change this check to 'AAA'.
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
