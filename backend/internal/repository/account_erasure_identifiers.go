// account_erasure_identifiers.go — Search remaining plaintext copies of an erased member's identifiers.
package repository

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/dflh-saf/backend/internal/model"
)

// Encrypted or hashed stores cannot hold plaintext identifiers.
var identifierExcludedTables = map[string]bool{
	"ALUMNI_ERASURE_CONTEXT":          true,
	"ALUMNI_DONATION_LEGAL_ARCHIVE":   true,
	"ALUMNI_ACCOUNT_DELETION_REQUEST": true,
}

const identifierMinRunes = 5

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// IdentifierMatches counts, per text column, rows that still contain any of
// the given identifiers. It only reads. Values shorter than five characters
// are ignored because they would match unrelated data.
func (r *AccountDeletionRequestRepository) IdentifierMatches(values []string) ([]model.AccountDeletionFootprint, error) {
	needles := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if utf8.RuneCountInString(value) >= identifierMinRunes && !seen[value] {
			seen[value] = true
			needles = append(needles, value)
		}
	}
	matches := []model.AccountDeletionFootprint{}
	if len(needles) == 0 {
		return matches, nil
	}
	var columns []struct {
		Table  string `db:"TABLE_NAME"`
		Column string `db:"COLUMN_NAME"`
	}
	err := r.DB.Select(&columns, `SELECT c.TABLE_NAME, c.COLUMN_NAME FROM information_schema.COLUMNS c
        JOIN information_schema.TABLES t ON t.TABLE_SCHEMA=c.TABLE_SCHEMA AND t.TABLE_NAME=c.TABLE_NAME
        WHERE c.TABLE_SCHEMA=DATABASE() AND t.TABLE_TYPE='BASE TABLE'
        AND c.DATA_TYPE IN ('char','varchar','tinytext','text','mediumtext','longtext')
        ORDER BY c.TABLE_NAME, c.COLUMN_NAME`)
	if err != nil {
		return nil, err
	}
	for _, col := range columns {
		if identifierExcludedTables[col.Table] || !deletionSQLIdentifier.MatchString(col.Table) || !deletionSQLIdentifier.MatchString(col.Column) {
			continue
		}
		conditions := make([]string, len(needles))
		args := make([]interface{}, len(needles))
		for i, needle := range needles {
			conditions[i] = fmt.Sprintf("`%s` LIKE ?", col.Column)
			args[i] = "%" + likeEscaper.Replace(needle) + "%"
		}
		var count int64
		// Tables and columns come from information_schema and are validated above.
		query := fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE %s", col.Table, strings.Join(conditions, " OR "))
		if err := r.DB.Get(&count, query, args...); err != nil {
			return nil, err
		}
		if count > 0 {
			matches = append(matches, model.AccountDeletionFootprint{Table: col.Table, Column: col.Column, Count: count})
		}
	}
	return matches, nil
}
