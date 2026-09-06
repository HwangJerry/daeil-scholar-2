// account_erasure_files.go — Delete only managed files; reject traversal and symlinks.
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type AccountErasureFiles struct{ UploadRoot, LegacyRoot, SiteOrigin string }

func (s *AccountErasureFiles) EraseURL(raw string) error {
	blocked := &model.ErasureBlocked{Code: "FILE_PATH_REVIEW_REQUIRED"}
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery != "" || u.Fragment != "" {
		return blocked
	}
	if u.IsAbs() || u.Host != "" {
		origin, e := url.Parse(s.SiteOrigin)
		if e != nil || origin.Host == "" || u.Host != origin.Host || u.Scheme != origin.Scheme {
			// Remote provider avatars are independent provider assets. The
			// external processor must resolve any app-owned remote uploads.
			return blocked
		}
	}
	root, relative := "", ""
	switch {
	case strings.HasPrefix(u.Path, "/uploads/"):
		root = s.UploadRoot
		relative = strings.TrimPrefix(u.Path, "/uploads/")
	case strings.HasPrefix(u.Path, "/files/"):
		root = s.LegacyRoot
		relative = strings.TrimPrefix(u.Path, "/files/")
	default:
		return blocked
	}
	if root == "" || relative == "" || strings.Contains(relative, "\\") || filepath.Clean(relative) != relative || strings.HasPrefix(relative, "../") {
		return blocked
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return blocked
	}
	// Refuse symlinks anywhere under the configured storage root, including the leaf.
	current := abs
	for _, part := range append([]string{""}, strings.Split(relative, "/")...) {
		current = filepath.Join(current, part)
		info, e := os.Lstat(current)
		if os.IsNotExist(e) {
			return nil
		}
		if e != nil {
			return blocked
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return blocked
		}
	}
	info, err := os.Stat(current)
	if err != nil {
		return blocked
	}
	if !info.Mode().IsRegular() {
		return blocked
	}
	if err = os.Remove(current); err != nil && !os.IsNotExist(err) {
		return &model.ErasureBlocked{Code: "FILE_DELETE_RETRY_REQUIRED"}
	}
	return nil
}
