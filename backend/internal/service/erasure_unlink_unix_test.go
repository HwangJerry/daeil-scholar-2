//go:build linux || darwin

// erasure_unlink_unix_test.go — A directory swap after inspection cannot redirect deletion.
package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestErasureUnlinkRejectsDirectorySwapAfterInspection(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "profile")
	if err = os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{filepath.Join(directory, "photo.jpg"), filepath.Join(outside, "photo.jpg")} {
		if err = os.WriteFile(file, []byte("synthetic"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	store := &AccountErasureFiles{UploadRoot: root}
	inspected, err := store.managedPath("/uploads/profile/photo.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(directory, filepath.Join(root, "old-profile")); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(outside, directory); err != nil {
		t.Fatal(err)
	}
	if unlinkErasureFile(inspected) == nil {
		t.Fatal("replaced directory accepted")
	}
	if _, err = os.Stat(filepath.Join(outside, "photo.jpg")); err != nil {
		t.Fatal("foreign file deleted")
	}
}
