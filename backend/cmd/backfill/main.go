// Backfill entry point — dispatches a task, connects to DB, and runs it.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func main() {
	if err := runBackfillCommand(context.Background(), os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func runBackfillCommand(ctx context.Context, arguments []string) error {
	command := "all"
	if len(arguments) > 0 {
		command = arguments[0]
		arguments = arguments[1:]
	}

	identityOptions := identityBackfillOptions{}
	switch command {
	case "identity":
		flags := flag.NewFlagSet("identity", flag.ContinueOnError)
		flags.SetOutput(os.Stderr)
		flags.BoolVar(&identityOptions.DryRun, "dry-run", false, "execute all writes in a transaction and roll it back")
		if err := flags.Parse(arguments); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("identity does not accept positional arguments: %v", flags.Args())
		}
	case "all", "summary", "thumbnail":
		if len(arguments) != 0 {
			return fmt.Errorf("%s does not accept arguments", command)
		}
	default:
		return errors.New("usage: backfill [all|summary|thumbnail|identity [--dry-run]]")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL env required")
	}
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	switch command {
	case "identity":
		_, err = BackfillIdentities(ctx, db, identityOptions)
		return err
	case "summary":
		BackfillSummaries(db)
	case "thumbnail":
		BackfillThumbnails(db)
	case "all":
		BackfillThumbnails(db)
		BackfillSummaries(db)
	}
	return nil
}
