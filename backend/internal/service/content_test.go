// content_test.go — Unit tests for content processing: HTML stripping, truncation, lazy loading, image extraction, encoding/decoding
package service

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text", "hello world", "hello world"},
		{"single tag", "<p>hello</p>", "hello"},
		{"nested tags", "<div><p>hello</p></div>", "hello"},
		{"img tag", `<img src="a.png">`, ""},
		{"entity amp", "a &amp; b", "a & b"},
		{"entity lt gt", "&lt;script&gt;", "<script>"},
		{"entity quot", `&quot;hi&quot;`, `"hi"`},
		{"entity apos", "it&#39;s", "it's"},
		{"newlines replaced", "line1\nline2", "line1 line2"},
		{"whitespace trimmed", "  hello  ", "hello"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripHTMLTags(tt.in)
			if got != tt.want {
				t.Errorf("StripHTMLTags(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"shorter than max", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"longer than max", "hello world", 5, "hello"},
		{"empty string", "", 5, ""},
		{"zero max", "hello", 0, ""},
		{"unicode truncation", "안녕하세요세계", 3, "안녕하"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateString(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestAddLazyLoading(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"adds loading=lazy",
			`<img src="a.png">`,
			`<img loading="lazy" src="a.png">`,
		},
		{
			"skips if already present",
			`<img loading="lazy" src="a.png">`,
			`<img loading="lazy" src="a.png">`,
		},
		{
			"multiple images",
			`<img src="a.png"><img src="b.png">`,
			`<img loading="lazy" src="a.png"><img loading="lazy" src="b.png">`,
		},
		{
			"no images",
			`<p>hello</p>`,
			`<p>hello</p>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addLazyLoading(tt.in)
			if got != tt.want {
				t.Errorf("addLazyLoading(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExtractFirstImageURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"has image", `<img src="https://example.com/a.png">`, "https://example.com/a.png"},
		{"multiple images returns first", `<img src="a.png"><img src="b.png">`, "a.png"},
		{"no image", `<p>hello</p>`, ""},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFirstImageURL(tt.in)
			if got != tt.want {
				t.Errorf("extractFirstImageURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDecodeContent(t *testing.T) {
	htmlStr := "<p>Hello World</p>"
	encoded := base64.StdEncoding.EncodeToString([]byte(htmlStr))

	tests := []struct {
		name     string
		contents string
		format   string
		want     string
	}{
		{"markdown format decodes base64", encoded, "MARKDOWN", htmlStr},
		{"non-markdown returns as-is", "raw content", "HTML", "raw content"},
		{"invalid base64 returns as-is", "not-valid-base64!!!", "MARKDOWN", "not-valid-base64!!!"},
		{"empty string markdown", "", "MARKDOWN", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeContent(tt.contents, tt.format)
			if got != tt.want {
				t.Errorf("DecodeContent(%q, %q) = %q, want %q", tt.contents, tt.format, got, tt.want)
			}
		})
	}
}

func TestConvertAndEncode(t *testing.T) {
	tests := []struct {
		name          string
		markdown      string
		wantInHTML    string // substring expected in decoded HTML
		wantSummary   string // substring expected in summary
		wantThumbLen  bool   // whether thumbnail should be non-empty
	}{
		{
			name:         "simple paragraph",
			markdown:     "Hello **World**",
			wantInHTML:   "Hello",
			wantSummary:  "Hello",
			wantThumbLen: false,
		},
		{
			name:         "with image",
			markdown:     `![alt](https://example.com/img.png)`,
			wantInHTML:   "img",
			wantSummary:  "",
			wantThumbLen: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, summary, thumbnail, err := ConvertAndEncode(tt.markdown)
			if err != nil {
				t.Fatalf("ConvertAndEncode(%q) returned error: %v", tt.markdown, err)
			}

			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Fatalf("failed to decode base64 result: %v", err)
			}
			html := string(decoded)

			if tt.wantInHTML != "" && !strings.Contains(html, tt.wantInHTML) {
				t.Errorf("decoded HTML %q does not contain %q", html, tt.wantInHTML)
			}
			if tt.wantSummary != "" && !strings.Contains(summary, tt.wantSummary) {
				t.Errorf("summary %q does not contain %q", summary, tt.wantSummary)
			}
			if tt.wantThumbLen && thumbnail == "" {
				t.Error("expected non-empty thumbnail, got empty")
			}
			if !tt.wantThumbLen && thumbnail != "" {
				t.Errorf("expected empty thumbnail, got %q", thumbnail)
			}
		})
	}
}
