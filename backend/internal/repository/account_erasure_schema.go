// account_erasure_schema.go — Explicit deletion targets with transactional schema checks.
package repository

import (
	"fmt"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type erasureSchema map[string]map[string]bool

func readErasureSchema(tx *sqlx.Tx) (erasureSchema, error) {
	var rows []struct {
		Table  string `db:"TABLE_NAME"`
		Column string `db:"COLUMN_NAME"`
	}
	err := tx.Select(&rows, `SELECT TABLE_NAME,COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE()`)
	s := erasureSchema{}
	for _, r := range rows {
		if s[r.Table] == nil {
			s[r.Table] = map[string]bool{}
		}
		s[r.Table][r.Column] = true
	}
	return s, err
}
func (s erasureSchema) has(t, c string) bool { return s[t][c] }
func (s erasureSchema) erase(tx *sqlx.Tx, table, where string, args ...interface{}) error {
	if s[table] == nil {
		return nil
	}
	// Tables and predicates are code-owned constants, never HTTP input.
	var engine string
	if err := tx.Get(&engine, `SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?`, table); err != nil {
		return err
	}
	if engine != "InnoDB" {
		return &model.ErasureBlocked{Code: "NON_TRANSACTIONAL_TABLE_" + table}
	}
	_, err := tx.Exec(fmt.Sprintf("DELETE FROM `%s` WHERE %s", table, where), args...)
	return err
}

// eraseAccountReferences applies accountReferenceSteps in order. The operator
// preview reads the same steps, so both always describe the same records.
func eraseAccountReferences(tx *sqlx.Tx, s erasureSchema, user int, email string) error {
	for _, step := range accountReferenceSteps(s, user, email) {
		if err := s.erase(tx, step.table, step.where, step.args...); err != nil {
			return err
		}
	}
	return nil
}
