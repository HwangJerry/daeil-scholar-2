// markdown_reencode — re-encodes all MARKDOWN posts after normalizing CONTENTS_MD
package migrator

import (
	"log"

	"github.com/jmoiron/sqlx"

	"github.com/dflh-saf/backend/internal/service"
)

// MarkdownPost holds a MARKDOWN-format board post for re-encoding.
type MarkdownPost struct {
	SEQ        int    `db:"SEQ"`
	ContentsMD string `db:"CONTENTS_MD"`
}

// ReencodeAllMarkdown normalizes CONTENTS_MD and re-encodes all MARKDOWN posts.
// When dryRun is true, the normalized markdown is logged but no DB writes occur.
func ReencodeAllMarkdown(db *sqlx.DB, dryRun bool) {
	posts, err := fetchAllMarkdownPosts(db)
	if err != nil {
		log.Fatalf("fetch markdown posts: %v", err)
	}
	log.Printf("found %d MARKDOWN posts to re-encode", len(posts))

	success, failed := 0, 0
	for _, post := range posts {
		normalized := NormalizeMarkdownLists(post.ContentsMD)

		encoded, summary, thumbnail, err := service.ConvertAndEncode(normalized)
		if err != nil {
			log.Printf("[SEQ %d] ConvertAndEncode failed: %v", post.SEQ, err)
			failed++
			continue
		}

		if dryRun {
			log.Printf("[DRY RUN][SEQ %d] normalized MD:\n%s", post.SEQ, normalized)
			success++
			continue
		}

		if err := updateMarkdownPost(db, post.SEQ, encoded, normalized, summary, thumbnail); err != nil {
			log.Printf("[SEQ %d] DB update failed: %v", post.SEQ, err)
			failed++
		} else {
			log.Printf("[SEQ %d] re-encoded", post.SEQ)
			success++
		}
	}

	log.Printf("re-encode complete — success: %d, failed: %d", success, failed)
}

func fetchAllMarkdownPosts(db *sqlx.DB) ([]MarkdownPost, error) {
	var posts []MarkdownPost
	err := db.Select(&posts, `
		SELECT SEQ, CONTENTS_MD
		FROM WEO_BOARDBBS
		WHERE CONTENT_FORMAT = 'MARKDOWN'
		  AND CONTENTS_MD IS NOT NULL
		  AND CONTENTS_MD != ''
		ORDER BY SEQ ASC
	`)
	return posts, err
}

func updateMarkdownPost(db *sqlx.DB, seq int, encoded, md, summary, thumbnail string) error {
	_, err := db.Exec(`
		UPDATE WEO_BOARDBBS
		SET CONTENTS      = ?,
		    CONTENTS_MD   = ?,
		    SUMMARY       = ?,
		    THUMBNAIL_URL = ?
		WHERE SEQ = ?
	`, encoded, md, summary, thumbnail, seq)
	return err
}
