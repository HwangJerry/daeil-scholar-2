// encode_helper — CLI: encode markdown via stdin→JSON, or run full LEGACY→MARKDOWN DB migration
// Modes:
//   encode         (default) read markdown from stdin, output {"encoded","summary","thumbnail"}
//   migrate-all    connect to DATABASE_URL and migrate all LEGACY posts
//   re-encode-all  normalize CONTENTS_MD and re-encode all MARKDOWN posts
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/dflh-saf/backend/internal/migrator"
	"github.com/dflh-saf/backend/internal/service"
)

// encodeResult is the JSON output for encode mode.
type encodeResult struct {
	Encoded   string `json:"encoded"`
	Summary   string `json:"summary"`
	Thumbnail string `json:"thumbnail"`
}

func main() {
	mode := "encode"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "encode":
		runEncode()
	case "migrate-all":
		runMigrateAll()
	case "re-encode-all":
		runReencodeAll()
	default:
		fmt.Fprintf(os.Stderr, "usage: encode_helper [encode|migrate-all|re-encode-all]\n")
		os.Exit(1)
	}
}

func runEncode() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("read stdin: %v", err)
	}

	encoded, summary, thumbnail, err := service.ConvertAndEncode(string(data))
	if err != nil {
		log.Fatalf("ConvertAndEncode: %v", err)
	}

	out := encodeResult{Encoded: encoded, Summary: summary, Thumbnail: thumbnail}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		log.Fatalf("json encode: %v", err)
	}
}

func runMigrateAll() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL env required (e.g. user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true)")
	}
	dryRun := os.Getenv("DRY_RUN") == "true"

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	migrator.MigrateAllLegacy(db, dryRun)
}

func runReencodeAll() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL env required (e.g. user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true)")
	}
	dryRun := os.Getenv("DRY_RUN") == "true"

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	migrator.ReencodeAllMarkdown(db, dryRun)
}
