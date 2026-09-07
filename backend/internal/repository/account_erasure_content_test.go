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
