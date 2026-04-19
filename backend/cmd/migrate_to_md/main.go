// migrate_to_md entry point — parses a SQL dump and generates a migration SQL file
package main

import (
	"log"
	"os"
)

func main() {
	dumpFile := os.Getenv("DUMP_FILE")
	if dumpFile == "" {
		dumpFile = "./WEO_BOARDBBS.sql"
	}
	sqlOutput := os.Getenv("SQL_OUTPUT")
	if sqlOutput == "" {
		sqlOutput = "./migrate_output.sql"
	}
	dryRun := os.Getenv("DRY_RUN") == "true"

	if dryRun {
		log.Println("[DRY RUN] No SQL file will be written")
	}

	posts, err := parseDump(dumpFile)
	if err != nil {
		log.Fatalf("parse dump: %v", err)
	}

	MigrateToMarkdown(posts, dryRun, sqlOutput)
}
