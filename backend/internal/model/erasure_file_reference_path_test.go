// erasure_file_reference_path_test.go — References are broader than safe unlink candidates.
package model

import "testing"

func TestSurvivingReferenceIdentity(t *testing.T) {
	for _, raw := range []string{
		"/files/shared.jpg?v=2#preview", "/upload/shared.jpg?", "files/shared.jpg",
		"https://www.app.example.org/files/shared.jpg", "//app.example.org/files/shared.jpg",
		"http://APP.EXAMPLE.ORG:80/old/upload/shared.jpg", "/files/sub/../shared.jpg",
		"/%66iles/shared.jpg", "/files/shared%2ejpg", " /files/shared.jpg?x=1&amp;y=2 ",
	} {
		local, external, err := ErasureFileReferencePath(raw, "https://app.example.org")
		if err != nil || external || local != "/files/shared.jpg" {
			t.Fatalf("%q: %q %v %v", raw, local, external, err)
		}
	}
	for _, raw := range []string{"", "data:image/png;base64,abc", "https://external.test/files/shared.jpg", "/assets/logo.png"} {
		local, _, err := ErasureFileReferencePath(raw, "https://app.example.org")
		if err != nil || local != "" {
			t.Fatal(raw, local, err)
		}
	}
	for _, raw := range []string{"/files/%zz", "/files/a\\b"} {
		if _, _, err := ErasureFileReferencePath(raw, "https://app.example.org"); err == nil {
			t.Fatal("malformed reference ignored", raw)
		}
	}
	local, _, err := ErasureFileReferencePath("/files/shared%252ejpg", "https://app.example.org")
	if err != nil || local != "/files/shared%252ejpg" {
		t.Fatal("double decoding", local, err)
	}
}
