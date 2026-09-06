// account_erasure_content.go — Discover managed inline files in stored post representations.
package repository

import (
	"encoding/base64"
	"github.com/jmoiron/sqlx"
	"html"
	"regexp"
	"strings"
)

var managedContentFile = regexp.MustCompile(`(?:https?://[^\s"'<>]+)?/(?:uploads|files)/[^\s"'<>\)\]]+`)

func managedContentURLs(value string) []string {
	values := []string{value}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		values = append(values, string(decoded))
	}
	urls := []string{}
	for _, v := range values {
		for _, url := range managedContentFile.FindAllString(html.UnescapeString(v), -1) {
			urls = append(urls, strings.TrimRight(url, ","))
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
			urls = append(urls, managedContentURLs(value)...)
		}
	}
	return urls, nil
}
