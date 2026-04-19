// migrate_update — generates SQL UPDATE statements for migrated posts
package main

import (
	"fmt"
	"strings"
)

type postUpdate struct {
	SEQ          int
	Contents     string
	ContentsMD   string
	Summary      string
	ThumbnailURL string
}

// generateUpdateSQL returns a single UPDATE statement for the given post.
// CONTENTS is base64-encoded HTML (no single quotes), so it needs no escaping.
// CONTENTS_MD and SUMMARY may contain single quotes and backslashes.
func generateUpdateSQL(u postUpdate) string {
	thumbnailSQL := "NULL"
	if u.ThumbnailURL != "" {
		thumbnailSQL = "'" + escapeSQLStr(u.ThumbnailURL) + "'"
	}
	return fmt.Sprintf(
		"UPDATE `WEO_BOARDBBS` SET `CONTENTS`='%s', `CONTENTS_MD`='%s', `CONTENT_FORMAT`='MARKDOWN', `SUMMARY`='%s', `THUMBNAIL_URL`=%s WHERE `SEQ`=%d;\n",
		u.Contents,
		escapeSQLStr(u.ContentsMD),
		escapeSQLStr(u.Summary),
		thumbnailSQL,
		u.SEQ,
	)
}

// escapeSQLStr escapes a string for use inside a single-quoted MySQL/MariaDB literal.
func escapeSQLStr(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\") // must come first
	s = strings.ReplaceAll(s, "'", "''")
	return s
}
