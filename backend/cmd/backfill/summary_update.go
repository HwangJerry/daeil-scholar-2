// Summary update — persists a computed summary for a single post
package main

import "github.com/jmoiron/sqlx"

func updateSummary(db *sqlx.DB, seq int, summary string) error {
	_, err := db.Exec(`UPDATE WEO_BOARDBBS SET SUMMARY = ? WHERE SEQ = ?`, summary, seq)
	return err
}
