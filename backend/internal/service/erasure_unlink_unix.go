//go:build linux || darwin

// erasure_unlink_unix.go — Pin parent directories while unlinking a managed file.
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"golang.org/x/sys/unix"
	"path/filepath"
	"strings"
)

// Re-open each ancestor without following links so a directory replaced after
// inspection cannot redirect unlink into another account's storage tree.
func unlinkErasureFile(absolute string) error {
	blocked := &model.ErasureBlocked{Code: "FILE_DELETE_RETRY_REQUIRED"}
	if !filepath.IsAbs(absolute) {
		return blocked
	}
	parts := strings.Split(strings.TrimPrefix(absolute, "/"), "/")
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return blocked
	}
	defer func() { unix.Close(fd) }()
	for _, part := range parts[:len(parts)-1] {
		next, err := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return blocked
		}
		unix.Close(fd)
		fd = next
	}
	leaf := parts[len(parts)-1]
	var stat unix.Stat_t
	err = unix.Fstatat(fd, leaf, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if err == unix.ENOENT {
		return nil
	}
	if err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return blocked
	}
	err = unix.Unlinkat(fd, leaf, 0)
	if err != nil && err != unix.ENOENT {
		return blocked
	}
	return nil
}
