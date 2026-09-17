// client_app_headers.go — Parsing of the version headers sent by the mobile apps.
package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
)

// Client version headers sent by the mobile apps on every API request. Browsers
// send none of these, so the web SPA is never evaluated.
//
// Debug builds deliberately omit them: their build number defaults to 1, and a
// request without headers is never gated (plan decision D13).
const (
	HeaderAppPlatform  = "X-App-Platform"
	HeaderAppBuild     = "X-App-Build"
	HeaderAppVersion   = "X-App-Version"
	HeaderAppOSVersion = "X-App-OS-Version"
)

type clientAppHeaders struct {
	Platform    string
	Build       int64
	VersionName string
	OSVersion   string
}

// readClientAppHeaders reports the client identity of a request, and whether it
// is usable at all. Anything malformed yields false so callers pass the request
// through untouched.
func readClientAppHeaders(r *http.Request) (clientAppHeaders, bool) {
	platform := strings.ToLower(strings.TrimSpace(r.Header.Get(HeaderAppPlatform)))
	if !model.IsSupportedAppPlatform(platform) {
		return clientAppHeaders{}, false
	}
	build, err := strconv.ParseInt(strings.TrimSpace(r.Header.Get(HeaderAppBuild)), 10, 64)
	if err != nil || build <= 0 {
		return clientAppHeaders{}, false
	}
	return clientAppHeaders{
		Platform:    platform,
		Build:       build,
		VersionName: r.Header.Get(HeaderAppVersion),
		OSVersion:   r.Header.Get(HeaderAppOSVersion),
	}, true
}
