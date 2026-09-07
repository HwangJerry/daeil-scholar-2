// erasure_file_reference_path.go — Conservative identity for surviving file references.
package model

import (
	"html"
	"net/url"
	"path"
	"strings"
)

// ErasureFileReferencePath identifies references, not authorized deletion paths.
// Query/fragment and public www/HTTP aliases can serve the same storage object.
// Invalid references fail closed; unrelated valid URLs have no managed identity.
func ErasureFileReferencePath(raw, siteOrigin string) (string, bool, error) {
	raw = strings.TrimSpace(html.UnescapeString(raw))
	if raw == "" {
		return "", false, nil
	}
	blocked := &ErasureBlocked{Code: "FILE_PATH_REVIEW_REQUIRED"}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false, blocked
	}
	if u.Host != "" {
		origin, err := url.Parse(siteOrigin)
		if err != nil || origin.Hostname() == "" {
			return "", false, blocked
		}
		host := func(s string) string { return strings.TrimPrefix(strings.TrimSuffix(strings.ToLower(s), "."), "www.") }
		if host(u.Hostname()) != host(origin.Hostname()) {
			return "", true, nil
		}
		if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
			return "", false, blocked
		}
	} else if u.IsAbs() {
		return "", true, nil
	}
	if strings.Contains(u.Path, "\\") {
		return "", false, blocked
	}
	// Resolve root-relative and relative managed URLs conservatively. Overmatching
	// retains a file for review; undermatching could destroy another user's file.
	clean := path.Clean("/" + strings.TrimLeft(u.Path, "/"))
	for _, prefix := range []string{"/uploads/", "/files/", "/upload/", "/old/upload/"} {
		if strings.HasPrefix(clean, prefix) && len(clean) > len(prefix) {
			canonical := "/files/"
			if prefix == "/uploads/" {
				canonical = prefix
			}
			return (&url.URL{Path: canonical + strings.TrimPrefix(clean, prefix)}).EscapedPath(), false, nil
		}
	}
	return "", false, nil
}
