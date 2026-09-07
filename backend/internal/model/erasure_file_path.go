// erasure_file_path.go — Canonical storage identity, separate from external URLs.
package model

import (
	"net/url"
	"path"
	"strings"
)

// ErasureFilePath returns a canonical local URL or an external URL handoff.
// External does not mean erased or exempt: it requires external-data evidence.
func ErasureFilePath(raw, siteOrigin string) (local string, external bool, err error) {
	blocked := &ErasureBlocked{Code: "FILE_PATH_REVIEW_REQUIRED"}
	u, e := url.Parse(raw)
	if e != nil || u.User != nil || u.Fragment != "" || strings.Contains(u.Path, "\\") {
		return "", false, blocked
	}
	if u.Host != "" || u.IsAbs() {
		if (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return "", false, blocked
		}
		origin, e := url.Parse(siteOrigin)
		if e != nil || origin.Host == "" || !strings.EqualFold(u.Host, origin.Host) || u.Scheme != origin.Scheme {
			return "", true, nil
		}
	}
	if u.RawQuery != "" || u.ForceQuery || path.Clean(u.Path) != u.Path {
		return "", false, blocked
	}
	for _, prefix := range []string{"/uploads/", "/files/", "/upload/", "/old/upload/"} {
		if strings.HasPrefix(u.Path, prefix) && len(u.Path) > len(prefix) {
			canonical := "/files/"
			if prefix == "/uploads/" {
				canonical = prefix
			}
			result := &url.URL{Path: canonical + strings.TrimPrefix(u.Path, prefix)}
			return result.EscapedPath(), false, nil
		}
	}
	return "", false, blocked
}
