// erasure_file_path_test.go — Shared aliases and foreign URLs have distinct storage identities.
package model

import "testing"

func TestErasureFileIdentity(t *testing.T) {
	for _, raw := range []string{"/files/shared.jpg", "/upload/shared.jpg", "/old/upload/shared.jpg", "https://app.example.org/files/shared.jpg"} {
		local, external, err := ErasureFilePath(raw, "https://app.example.org")
		if err != nil || external || local != "/files/shared.jpg" {
			t.Fatal(raw, local, external, err)
		}
	}
	if _, external, err := ErasureFilePath("https://profile.example.net/avatar.jpg?size=large", "https://app.example.org"); err != nil || !external {
		t.Fatal("external handoff rejected")
	}
	for _, raw := range []string{"/files/../private", "/files/%2e%2e/private", "javascript:bad", "//foreign.example/files/a", "/files/a?x=y"} {
		if _, _, err := ErasureFilePath(raw, "https://app.example.org"); err == nil {
			t.Fatal("unsafe path accepted", raw)
		}
	}
}
