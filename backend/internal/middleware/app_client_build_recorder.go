// app_client_build_recorder.go — Records which app builds authenticated users run.
package middleware

import "net/http"

type AppClientBuildObserver interface {
	Observe(platform string, build int64, versionName string)
}

// AppClientBuildRecorder notes the build behind a request so administrators can
// pick a real build number when setting update thresholds.
//
// It records only authenticated requests. The recorded maximum bounds the
// thresholds an administrator may set, so an anonymous caller must not be able
// to introduce a build number — nor to grow the table at will.
func AppClientBuildRecorder(observer AppClientBuildObserver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if observer == nil || GetAuthUser(r.Context()) == nil {
				next.ServeHTTP(w, r)
				return
			}
			if headers, ok := readClientAppHeaders(r); ok {
				observer.Observe(headers.Platform, headers.Build, headers.VersionName)
			}
			next.ServeHTTP(w, r)
		})
	}
}
