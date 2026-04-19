// legacy_migrator — fetches LEGACY posts and migrates them to MARKDOWN format in the DB
package migrator

import (
	"log"

	"github.com/jmoiron/sqlx"

	"github.com/dflh-saf/backend/internal/converter"
	"github.com/dflh-saf/backend/internal/service"
)

// LegacyPost holds a LEGACY-format board post fetched from the DB.
type LegacyPost struct {
	SEQ      int    `db:"SEQ"`
	Gate     string `db:"GATE"`
	Subject  string `db:"SUBJECT"`
	Contents string `db:"CONTENTS"`
}

// MigrateAllLegacy fetches all LEGACY posts across all gates and migrates each one.
// When dryRun is true, transformations are logged but no DB writes occur.
func MigrateAllLegacy(db *sqlx.DB, dryRun bool) {
	posts, err := fetchAllLegacyPosts(db)
	if err != nil {
		log.Fatalf("fetch legacy posts: %v", err)
	}
	log.Printf("found %d LEGACY posts to migrate", len(posts))

	success, failed := 0, 0
	for _, post := range posts {
		md := converter.FromLegacyPost(post.Subject, post.Contents)

		encoded, summary, thumbnail, err := service.ConvertAndEncode(md)
		if err != nil {
			log.Printf("[SEQ %d] ConvertAndEncode failed: %v", post.SEQ, err)
			failed++
			continue
		}

		if dryRun {
			log.Printf("[DRY RUN][SEQ %d][%s] %q → summary=%q", post.SEQ, post.Gate, post.Subject, summary)
			success++
			continue
		}

		if err := updatePost(db, post.SEQ, encoded, md, summary, thumbnail); err != nil {
			log.Printf("[SEQ %d] DB update failed: %v", post.SEQ, err)
			failed++
		} else {
			log.Printf("[SEQ %d][%s] migrated: %q", post.SEQ, post.Gate, post.Subject)
			success++
		}
	}

	log.Printf("migration complete — success: %d, failed: %d", success, failed)
}

func fetchAllLegacyPosts(db *sqlx.DB) ([]LegacyPost, error) {
	var posts []LegacyPost
	err := db.Select(&posts, `
		SELECT SEQ, GATE, SUBJECT, CONTENTS
		FROM WEO_BOARDBBS
		WHERE CONTENT_FORMAT = 'LEGACY'
		  AND CONTENTS IS NOT NULL
		  AND CONTENTS != ''
		ORDER BY SEQ ASC
	`)
	return posts, err
}

func updatePost(db *sqlx.DB, seq int, encoded, md, summary, thumbnail string) error {
	_, err := db.Exec(`
		UPDATE WEO_BOARDBBS
		SET CONTENTS        = ?,
		    CONTENTS_MD     = ?,
		    CONTENT_FORMAT  = 'MARKDOWN',
		    SUMMARY         = ?,
		    THUMBNAIL_URL   = ?
		WHERE SEQ = ?
	`, encoded, md, summary, thumbnail, seq)
	return err
}
