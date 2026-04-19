// rerender_md — re-encodes all MARKDOWN posts with the updated goldmark renderer
// Run: DATABASE_URL=... go run ./cmd/rerender_md
package main

import (
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/dflh-saf/backend/internal/service"
)

type post struct {
	SEQ        int    `db:"SEQ"`
	ContentsMD string `db:"CONTENTS_MD"`
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL env required (e.g. user:pass@tcp(host:3306)/dbname?charset=utf8mb4&parseTime=true)")
	}

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var posts []post
	if err := db.Select(&posts, `
		SELECT SEQ, CONTENTS_MD
		FROM WEO_BOARDBBS
		WHERE CONTENT_FORMAT = 'MARKDOWN'
		  AND CONTENTS_MD != ''
	`); err != nil {
		log.Fatalf("select: %v", err)
	}
	log.Printf("found %d MARKDOWN posts to re-render", len(posts))

	ok, fail := 0, 0
	for _, p := range posts {
		encoded, summary, thumbnail, err := service.ConvertAndEncode(p.ContentsMD)
		if err != nil {
			log.Printf("SEQ=%d convert error: %v", p.SEQ, err)
			fail++
			continue
		}
		if _, err := db.Exec(`
			UPDATE WEO_BOARDBBS
			SET CONTENTS      = ?,
			    SUMMARY       = ?,
			    THUMBNAIL_URL = ?
			WHERE SEQ = ?
		`, encoded, summary, thumbnail, p.SEQ); err != nil {
			log.Printf("SEQ=%d update error: %v", p.SEQ, err)
			fail++
			continue
		}
		ok++
	}

	log.Printf("done — ok=%d fail=%d", ok, fail)
}
