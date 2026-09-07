// account_erasure_content_test.go — Legacy inline uploads remain discoverable after HTML storage.
package repository

import (
	"encoding/base64"
	"github.com/dflh-saf/backend/internal/model"
	"testing"
)

func TestErasureDiscoversLegacyInlineImagesFromEncodedHTML(t *testing.T) {
	content := base64.StdEncoding.EncodeToString([]byte(`<img src="/upload/board/old.jpg"><img src="/old/upload/board/thumb.jpg">`))
	urls := managedContentURLs(content)
	if len(urls) != 2 || urls[0] != "/upload/board/old.jpg" || urls[1] != "/old/upload/board/thumb.jpg" {
		t.Fatal("legacy inline files lost", urls)
	}
}

func TestSurvivingContentReferenceVariants(t *testing.T) {
	for _, value := range []string{
		`<img src=/%66iles/shared.jpg?v=1>`, `<img src="files/shared.jpg#preview">`,
		`![image](/%66iles/shared.jpg)`, `/%66iles/shared.jpg`,
		base64.StdEncoding.EncodeToString([]byte(`<img src="/%66iles/shared.jpg">`)),
	} {
		found := false
		for _, raw := range survivingContentURLs(value) {
			local, _, err := model.ErasureFileReferencePath(raw, "https://app.example.org")
			if err == nil && local == "/files/shared.jpg" {
				found = true
			}
		}
		if !found {
			t.Fatal("surviving reference missed", value)
		}
	}
}

func TestErasureContentDecodesOnlyHTMLAttributes(t *testing.T) {
	for _, c := range []struct{ name, content, want string }{
		{"raw URL", "/files/a&copy;.jpg", "/files/a&copy;.jpg"},
		{"HTML", `<img src="/files/a&amp;copy;.jpg">`, "/files/a&copy;.jpg"},
		{"encoded HTML", base64.StdEncoding.EncodeToString([]byte(`<img src="/files/a&amp;copy;.jpg">`)), "/files/a&copy;.jpg"},
		{"quoted filename", `<img src="/files/a&quot;b.jpg">`, `/files/a"b.jpg`},
		{"Markdown", `![photo](/files/a&copy;.jpg)`, "/files/a&copy;.jpg"},
	} {
		t.Run(c.name, func(t *testing.T) {
			urls := managedContentURLs(c.content)
			if len(urls) != 1 || urls[0] != c.want {
				t.Fatalf("identity changed: got %q want %q", urls, c.want)
			}
		})
	}
}

func TestAmbiguousBrowserReferencesRemainDiscoverable(t *testing.T) {
	for _, raw := range []string{"https:/files/shared.jpg", "https:files/shared.jpg", "HTTPS:files/shared.jpg", "https:////app.example.org/files/shared.jpg", "///app.example.org/files/shared.jpg"} {
		for _, content := range []string{
			`<img src="` + raw + `">`, `<a href="` + raw + `">image</a>`,
			`<img srcset="` + raw + ` 1x">`, `<div style="background-image:url(` + raw + `)"></div>`,
			base64.StdEncoding.EncodeToString([]byte(`<img src="` + raw + `">`)),
		} {
			found := false
			for _, candidate := range survivingContentURLs(content) {
				local, _, err := model.ErasureFileReferencePath(candidate, "https://app.example.org")
				if err != nil || local == "/files/shared.jpg" {
					found = true
				}
			}
			if !found {
				t.Fatalf("ambiguous reference was missed: %q", content)
			}
		}
	}
}

func TestHTTPDiscoveryDoesNotQueueUnmanagedLinks(t *testing.T) {
	content := `<a href="https://external.test/donate">donate</a>
<img src="https://external.test/assets/logo.png">
<a href="https://app.example.org/privacy">privacy</a>
https://external.test/news
https://app.example.org/assets/logo.png`
	if urls := managedContentURLs(content); len(urls) != 0 {
		t.Fatalf("unmanaged links became deletion candidates: %q", urls)
	}
	for _, raw := range survivingContentURLs(content) {
		local, _, err := model.ErasureFileReferencePath(raw, "https://app.example.org")
		if err != nil || local != "" {
			t.Fatalf("unmanaged link blocks local deletion: %q %q %v", raw, local, err)
		}
	}
}
