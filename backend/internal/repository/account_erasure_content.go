// account_erasure_content.go — Discover managed inline files without changing raw URL identities.
package repository

import (
	"encoding/base64"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	"golang.org/x/net/html"
)

// Keep complete HTTP(S) tokens even when net/url cannot recover their host.
// Taking only a /files suffix (or missing an opaque https:files URL) can lose
// the browser's identity; the reference normalizer decides whether to block.
var managedContentFile = regexp.MustCompile(`(?i)(?:https?:[^\s"'<>\)\]]+|(?://[^\s"'<>]+)?/(?:uploads|files|upload|old/upload)/[^\s"'<>\)\]]+)`)
var contentMarkdownURL = regexp.MustCompile(`\]\(<?([^\s)>]+)>?(?:\s+[^)]*)?\)`)

// HTML token attributes are entity-decoded exactly once by the tokenizer. Raw
// profile, thumbnail, Markdown and file-list URLs must never be HTML-decoded.
func survivingContentURLs(value string) []string {
	values := []string{value}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		values = append(values, string(decoded))
	}
	urls := []string{}
	seen := map[string]bool{}
	add := func(raw string) {
		if raw != "" && !seen[raw] {
			urls = append(urls, raw)
			seen[raw] = true
		}
	}
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if !strings.ContainsAny(trimmed, " \t\r\n<>\"'") {
			for _, prefix := range []string{"/", "files/", "upload/", "uploads/", "old/upload/", "https://", "http://"} {
				if strings.HasPrefix(trimmed, prefix) {
					add(trimmed)
					break
				}
			}
		}
		tokenizer := html.NewTokenizer(strings.NewReader(v))
		for {
			kind := tokenizer.Next()
			if kind == html.ErrorToken {
				// A tokenizer can still expose its final text before EOF. Non-EOF errors
				// leave only conservative raw matches, never a decoded deletion identity.
				if tokenizer.Err() != io.EOF {
					add(v)
				}
				break
			}
			switch kind {
			case html.StartTagToken, html.SelfClosingTagToken:
				for _, attr := range tokenizer.Token().Attr {
					if attr.Key == "src" || attr.Key == "href" || attr.Key == "poster" {
						add(attr.Val)
					} else {
						// Preserve conservative discovery in legacy srcset, data and
						// style attributes, which have already been decoded once.
						for _, raw := range managedContentFile.FindAllString(attr.Val, -1) {
							add(strings.TrimRight(raw, ","))
						}
					}
				}
			case html.TextToken:
				// Raw() preserves literal ampersands in Markdown and unstructured URL lists.
				text := string(tokenizer.Raw())
				for _, raw := range managedContentFile.FindAllString(text, -1) {
					add(strings.TrimRight(raw, ","))
				}
				for _, match := range contentMarkdownURL.FindAllStringSubmatch(text, -1) {
					add(match[1])
				}
			}
		}
	}
	return urls
}

func managedContentURLs(value string) []string {
	urls := []string{}
	for _, raw := range survivingContentURLs(value) {
		u, err := url.Parse(raw)
		if err != nil {
			if managedContentFile.MatchString(raw) {
				urls = append(urls, raw)
			}
			continue
		}
		clean := path.Clean("/" + strings.TrimLeft(u.Path, "/"))
		for _, prefix := range []string{"/uploads/", "/files/", "/upload/", "/old/upload/"} {
			if strings.HasPrefix(clean, prefix) {
				urls = append(urls, raw)
				break
			}
		}
	}
	return urls
}

func postErasureURLs(tx *sqlx.Tx, s erasureSchema, user int) ([]string, error) {
	urls := []string{}
	for _, col := range []string{"CONTENTS", "CONTENTS_MD", "THUMBNAIL_URL", "FILES", "RE_FILES"} {
		if !s.has("WEO_BOARDBBS", col) {
			continue
		}
		var values []string
		if err := tx.Select(&values, "SELECT COALESCE(`"+col+"`,'') FROM WEO_BOARDBBS WHERE USR_SEQ=?", user); err != nil {
			return nil, err
		}
		for _, value := range values {
			if col == "THUMBNAIL_URL" {
				if value != "" {
					urls = append(urls, value)
				}
			} else {
				urls = append(urls, managedContentURLs(value)...)
			}
		}
	}
	return urls, nil
}
