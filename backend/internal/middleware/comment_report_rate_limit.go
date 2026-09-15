package middleware

import (
	"github.com/patrickmn/go-cache"
	"net/http"
	"strconv"
	"time"
)

// Initial abuse budget matches login protection; one account bucket spans all comments.
const commentReportMaxAttempts = 10
const commentReportLimitWindow = 15 * time.Minute

func CommentReportRateLimiter() func(http.Handler) http.Handler {
	c := cache.New(commentReportLimitWindow, commentReportLimitWindow)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetAuthUser(r.Context())
			if user == nil {
				respondError(w, 401, "UNAUTHORIZED", "로그인이 필요합니다.")
				return
			}
			key := strconv.Itoa(user.USRSeq)
			c.Add(key, 0, commentReportLimitWindow)
			count, _ := c.IncrementInt(key, 1)
			if count > commentReportMaxAttempts {
				w.Header().Set("Retry-After", strconv.Itoa(int(commentReportLimitWindow/time.Second)))
				respondError(w, 429, "RATE_LIMITED", "잠시 후 다시 시도해주세요.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
