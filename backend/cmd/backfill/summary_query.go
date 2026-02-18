// Summary query — fetches posts that need summary backfill
package main

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type postRow struct {
	SEQ      int            `db:"SEQ"`
	Contents sql.NullString `db:"CONTENTS"`
}

func fetchPostsWithoutSummary(db *sqlx.DB) ([]postRow, error) {
	var posts []postRow
	err := db.Select(&posts, `
		SELECT SEQ, CONTENTS FROM WEO_BOARDBBS
		WHERE GATE = 'NOTICE' AND (SUMMARY IS NULL OR SUMMARY = '')
		  AND CONTENTS IS NOT NULL AND CONTENTS != ''
	`)
	return posts, err
}
