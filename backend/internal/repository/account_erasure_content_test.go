// account_erasure_content_test.go — Legacy inline uploads remain discoverable after HTML storage.
package repository

import (
	"encoding/base64"
	"testing"
)

func TestErasureDiscoversLegacyInlineImagesFromEncodedHTML(t *testing.T) {
	content := base64.StdEncoding.EncodeToString([]byte(`<img src="/upload/board/old.jpg"><img src="/old/upload/board/thumb.jpg">`))
	urls := managedContentURLs(content)
	if len(urls) != 2 || urls[0] != "/upload/board/old.jpg" || urls[1] != "/old/upload/board/thumb.jpg" {
		t.Fatal("legacy inline files lost", urls)
	}
}
