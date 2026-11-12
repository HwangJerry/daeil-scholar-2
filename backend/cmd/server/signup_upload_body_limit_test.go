package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIBodyLimitOnlyRaisesSignupPhotoPOST(t *testing.T) {
	for _, c := range []struct {
		name, path, method string
		size               int
		allowed            bool
	}{
		{"photo above 2MiB", "/api/auth/social/link/photo", "POST", 3 << 20, true},
		{"photo with multipart overhead", "/api/auth/social/link/photo", "POST", (5 << 20) + 1024, true},
		{"photo above envelope", "/api/auth/social/link/photo", "POST", int(signupPhotoRequestMaxBytes) + 1, false},
		{"json stays bounded", "/api/auth/register", "POST", 3 << 20, false},
		{"other method stays bounded", "/api/auth/social/link/photo", "GET", 3 << 20, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			var err error
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, err = io.ReadAll(r.Body) })
			req := httptest.NewRequest(c.method, c.path, bytes.NewReader(make([]byte, c.size)))
			apiBodySizeLimit(next).ServeHTTP(httptest.NewRecorder(), req)
			if (err == nil) != c.allowed {
				t.Fatalf("allowed=%v error=%v", c.allowed, err)
			}
		})
	}
}
