// app_update_policy_repo.go — MariaDB persistence for app update policies and their audit trail.
package repository

import (
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AppUpdatePolicyRepository struct {
	db *sqlx.DB
}

func NewAppUpdatePolicyRepository(db *sqlx.DB) *AppUpdatePolicyRepository {
	return &AppUpdatePolicyRepository{db: db}
}

// GetPolicySetting reads one policy row. A missing row surfaces as sql.ErrNoRows.
func (r *AppUpdatePolicyRepository) GetPolicySetting(key string) (model.AppSetting, error) {
	var setting model.AppSetting
	err := r.db.Get(&setting, `
		SELECT AS_KEY, AS_VALUE, AS_DESCRIPTION, AS_PUBLIC, UPDATED_AT, UPDATED_BY
		FROM app_settings
		WHERE AS_KEY = ?
	`, key)
	return setting, err
}

// SavePolicy replaces the policy and records the change in one transaction.
//
// The row is locked and its UPDATED_AT compared with the value the administrator
// read; a mismatch reports a conflict and writes nothing, so a concurrent edit is
// never silently overwritten. A zero expectedUpdatedAt skips that check.
func (r *AppUpdatePolicyRepository) SavePolicy(
	key, platform, afterJSON string,
	updatedBy int,
	expectedUpdatedAt time.Time,
) (bool, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	var current struct {
		Value     string    `db:"AS_VALUE"`
		UpdatedAt time.Time `db:"UPDATED_AT"`
	}
	if err := tx.Get(&current, `
		SELECT AS_VALUE, UPDATED_AT
		FROM app_settings
		WHERE AS_KEY = ?
		FOR UPDATE
	`, key); err != nil {
		return false, err
	}
	if !expectedUpdatedAt.IsZero() && !current.UpdatedAt.Equal(expectedUpdatedAt) {
		return true, nil
	}

	if _, err := tx.Exec(`
		UPDATE app_settings
		SET AS_VALUE = ?, UPDATED_AT = NOW(), UPDATED_BY = ?
		WHERE AS_KEY = ?
	`, afterJSON, updatedBy, key); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`
		INSERT INTO app_update_policy_history (
			PLATFORM, BEFORE_JSON, AFTER_JSON, CHANGED_BY, CHANGED_AT
		) VALUES (?, ?, ?, ?, NOW())
	`, platform, current.Value, afterJSON, updatedBy); err != nil {
		return false, err
	}

	return false, tx.Commit()
}

// ListHistory returns recent policy changes for one platform, newest first.
func (r *AppUpdatePolicyRepository) ListHistory(platform string, limit int) ([]model.AppUpdatePolicyHistoryEntry, error) {
	entries := make([]model.AppUpdatePolicyHistoryEntry, 0)
	err := r.db.Select(&entries, `
		SELECT AUPH_SEQ, PLATFORM, BEFORE_JSON, AFTER_JSON, CHANGED_BY, CHANGED_AT
		FROM app_update_policy_history
		WHERE PLATFORM = ?
		ORDER BY CHANGED_AT DESC, AUPH_SEQ DESC
		LIMIT ?
	`, platform, limit)
	return entries, err
}
