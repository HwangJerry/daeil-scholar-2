// Backfill entry point — connects to DB and runs backfill tasks
package main

import (
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL env required")
	}
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	BackfillThumbnails(db)
	BackfillSummaries(db)
}
