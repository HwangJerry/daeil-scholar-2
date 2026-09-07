// erasure_file_reference_path.go — Conservative identity for surviving file references.
package model

import (
	"net/url"
	"path"
	"strings"
)

// ErasureFileReferencePath identifies references, not authorized deletion paths.
// Query/fragment and public www/HTTP aliases can serve the same storage object.
// Invalid references fail closed; unrelated valid URLs have no managed identity.
func ErasureFileReferencePath(raw, siteOrigin string) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, nil
	}
	blocked := &ErasureBlocked{Code: "FILE_PATH_REVIEW_REQUIRED"}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false, blocked
	}
	// With three or more leading slashes browsers recover an authority, while
	// net/url leaves the whole value in Path. Its storage identity is ambiguous.
	if u.Host == "" && strings.HasPrefix(raw, "//") {
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
		// Browsers resolve hostless HTTP(S) URLs such as https:/files/a.jpg
		// against the page, or recover a host from extra slashes. net/url does
		// neither. Without the page URL, these must require review instead of
		// being declared external and allowing a shared file to be unlinked.
		if strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https") {
			return "", false, blocked
		}
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
