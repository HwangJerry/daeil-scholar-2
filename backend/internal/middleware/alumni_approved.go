package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/service"
)

func AlumniApprovedMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claim := GetAuthUser(r.Context())
			if claim == nil || claim.USRSeq <= 0 {
				respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
				return
			}
			current, err := authService.GetCurrentUser(claim.USRSeq)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "AUTH_PRINCIPAL_FAILED", "사용자 권한을 확인하지 못했습니다")
				return
			}
			if current == nil {
				respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "유효하지 않은 세션입니다")
				return
			}
			if err := (service.LoginEligibilityPolicy{}).EnsureStatusAllowed(current.USRStatus); err != nil {
				respondError(w, http.StatusForbidden, "ACCOUNT_INELIGIBLE", "현재 계정 상태에서는 접근할 수 없습니다")
				return
			}
			if current.Verification.Status != "approved" {
				status := string(current.Verification.Status)
				if status == "" {
					status = "unverified"
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    "ALUMNI_APPROVAL_REQUIRED",
					"message": "동문 인증 승인 후 이용할 수 있습니다.",
					"details": map[string]string{"verificationStatus": status},
				})
				return
			}
			ctx := SetAuthUser(r.Context(), current)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
