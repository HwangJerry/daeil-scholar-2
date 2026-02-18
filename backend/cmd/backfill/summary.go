// Summary backfill — orchestrates summary generation for notices missing one
package main

import (
	"log"

	"github.com/jmoiron/sqlx"
)

func BackfillSummaries(db *sqlx.DB) {
	posts, err := fetchPostsWithoutSummary(db)
	if err != nil {
		log.Printf("summary query error: %v", err)
		return
	}

	count := 0
	for _, p := range posts {
		if !p.Contents.Valid {
			continue
		}
		summary := extractSummary(p.Contents.String)
		if summary == "" {
			continue
		}
		if err := updateSummary(db, p.SEQ, summary); err != nil {
			log.Printf("failed to update summary for SEQ %d: %v", p.SEQ, err)
			continue
		}
		count++
	}
	log.Printf("backfilled %d summaries", count)
}
