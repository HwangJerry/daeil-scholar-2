// erasure_file_reference_path_test.go — References are broader than safe unlink candidates.
package model

import (
	"errors"
	"testing"
)

func TestHostlessHTTPReferencesRequireReview(t *testing.T) {
	for _, raw := range []string{
		"https:/files/shared.jpg", "https:files/shared.jpg",
		"https:///app.example.org/files/shared.jpg", "https:////app.example.org/files/shared.jpg",
		"http:/files/shared.jpg", "HTTP:files/shared.jpg", "HTTPS:/files/shared.jpg?x=1#preview",
		"///app.example.org/files/shared.jpg", "////app.example.org/files/shared.jpg",
		"https:\\app.example.org/files/shared.jpg", "https://app.example.org\\@external.test/files/shared.jpg",
		"https://app.exa\tmple.org/files/shared.jpg", "https://external.test\n@app.example.org/files/shared.jpg",
	} {
		t.Run(raw, func(t *testing.T) {
			local, external, err := ErasureFileReferencePath(raw, "https://app.example.org")
			var blocked *ErasureBlocked
			if !errors.As(err, &blocked) || blocked.Code != "FILE_PATH_REVIEW_REQUIRED" || external || local != "" {
				t.Fatalf("ambiguous browser URL ignored: local=%q external=%v err=%v", local, external, err)
			}
		})
	}
}

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

func TestRawReferencePreservesEntityLikeFileNames(t *testing.T) {
	for _, raw := range []string{"/files/a&copy;.jpg", "/files/a&amp;.jpg", "/files/a%26copy%3B.jpg"} {
		expected, _, err := ErasureFilePath(raw, "https://app.example.org")
		if err != nil {
			t.Fatal(err)
		}
		actual, external, err := ErasureFileReferencePath(raw, "https://app.example.org")
		if err != nil || external || actual != expected {
			t.Fatalf("raw URL changed: %q -> %q, wanted %q, %v", raw, actual, expected, err)
		}
	}
}
