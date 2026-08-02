// approved_alumni.go — Authorization middleware for approved-alumni-only features.
package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
)

func ApprovedAlumniMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetAuthUser(r.Context())
		if user == nil {
			respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
			return
		}
		if user.Verification.Status != model.VerificationApproved {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(model.APIError{
				Code:    "ALUMNI_APPROVAL_REQUIRED",
				Message: "동문 인증 승인 후 이용할 수 있습니다.",
				Details: map[string]model.VerificationStatus{
					"verificationStatus": user.Verification.Status,
				},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
