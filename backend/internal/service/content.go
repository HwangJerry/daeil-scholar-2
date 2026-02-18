package service

import (
	"bytes"
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // tables, strikethrough, autolinks
			highlighting.NewHighlighting(highlighting.WithStyle("monokai")),
		),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)

	sanitizer = func() *bluemonday.Policy {
		p := bluemonday.UGCPolicy()
		p.AllowImages()
		p.AllowAttrs("loading").Matching(regexp.MustCompile(`^lazy$`)).OnElements("img")
		p.AllowAttrs("class").Matching(regexp.MustCompile(`^(chroma|language-\w+|highlight).*$`)).Globally()
		return p
	}()

	imgTagRe  = regexp.MustCompile(`<img([^>]*)>`)
	imgSrcRe  = regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
)

// ConvertAndEncode converts Markdown to sanitized HTML, then Base64-encodes it.
// Returns (encoded HTML, summary, thumbnail URL, error).
func ConvertAndEncode(markdownText string) (string, string, string, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(markdownText), &buf); err != nil {
		return "", "", "", err
	}
	rawHTML := buf.String()

	rawHTML = addLazyLoading(rawHTML)

	safeHTML := sanitizer.Sanitize(rawHTML)

	encoded := base64.StdEncoding.EncodeToString([]byte(safeHTML))

	plain := StripHTMLTags(safeHTML)
	summary := TruncateString(plain, 200)

	thumbnail := extractFirstImageURL(safeHTML)

	return encoded, summary, thumbnail, nil
}

// DecodeContent decodes DB content to HTML based on the content format.
func DecodeContent(contents string, format string) string {
	if format == "MARKDOWN" {
		decoded, err := base64.StdEncoding.DecodeString(contents)
		if err != nil {
			return contents
		}
		return string(decoded)
	}
	return contents
}

func addLazyLoading(h string) string {
	return imgTagRe.ReplaceAllStringFunc(h, func(match string) string {
		if strings.Contains(match, "loading=") {
			return match
		}
		return strings.Replace(match, "<img", `<img loading="lazy"`, 1)
	})
}

func extractFirstImageURL(h string) string {
	matches := imgSrcRe.FindStringSubmatch(h)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// StripHTMLTags removes all HTML tags and decodes common HTML entities.
func StripHTMLTags(h string) string {
	text := htmlTagRe.ReplaceAllString(h, "")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", `"`)
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)
	return text
}

// TruncateString returns the first maxLen runes of s.
func TruncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
