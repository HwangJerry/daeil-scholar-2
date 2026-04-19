// migrate_query — fetches LEGACY-format posts that need markdown migration
package main

import "github.com/jmoiron/sqlx"

type legacyPost struct {
	SEQ      int    `db:"SEQ"`
	Subject  string `db:"SUBJECT"`
	Contents string `db:"CONTENTS"`
}

func fetchLegacyPosts(db *sqlx.DB) ([]legacyPost, error) {
	var posts []legacyPost
	err := db.Select(&posts, `
		SELECT SEQ, SUBJECT, CONTENTS
		FROM WEO_BOARDBBS
		WHERE GATE = 'NOTICE'
		  AND OPEN_YN = 'Y'
		  AND CONTENT_FORMAT = 'LEGACY'
		  AND CONTENTS IS NOT NULL
		  AND CONTENTS != ''
		ORDER BY SEQ ASC
	`)
	return posts, err
}
