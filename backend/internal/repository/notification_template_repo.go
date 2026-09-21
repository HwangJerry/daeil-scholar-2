// notification_template_repo.go — MariaDB persistence for admin-editable notification texts.
package repository

import (
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// ErrNotificationTemplateConflict reports that the row changed since the
// administrator loaded it, so the submitted edit was not written.
var ErrNotificationTemplateConflict = errors.New("notification template changed since it was read")

type NotificationTemplateRepository struct {
	db *sqlx.DB
}

func NewNotificationTemplateRepository(db *sqlx.DB) *NotificationTemplateRepository {
	return &NotificationTemplateRepository{db: db}
}

// ListAll returns every stored override, ordered by key.
func (r *NotificationTemplateRepository) ListAll() ([]model.NotificationTemplate, error) {
	templates := make([]model.NotificationTemplate, 0)
	err := r.db.Select(&templates, `
		SELECT NT_KEY, NT_CHANNEL, NT_TITLE, NT_BODY, NT_VERSION, UPDATED_AT, UPDATED_BY
		FROM notification_templates
		ORDER BY NT_KEY ASC
	`)
	return templates, err
}

// Get reads one stored override. A missing row is not an error: the catalog
// default applies, so the caller receives (nil, nil).
func (r *NotificationTemplateRepository) Get(key string) (*model.NotificationTemplate, error) {
	var template model.NotificationTemplate
	err := r.db.Get(&template, `
		SELECT NT_KEY, NT_CHANNEL, NT_TITLE, NT_BODY, NT_VERSION, UPDATED_AT, UPDATED_BY
		FROM notification_templates
		WHERE NT_KEY = ?
	`, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// Update writes one override and returns the version it produced.
//
// The row is locked for the length of the transaction and its version compared
// with the one the administrator was editing, so a concurrent save is reported
// as ErrNotificationTemplateConflict rather than silently overwritten. There is
// no way to skip that check: expectedVersion must be positive, and since stored
// versions start at 1, a non-positive value is reported as a conflict like any
// other mismatch. A missing row surfaces as sql.ErrNoRows, because keys come
// from a catalog fixed in code and cannot be created here.
func (r *NotificationTemplateRepository) Update(
	key, title, body string,
	updatedBy int,
	expectedVersion int,
) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var currentVersion int
	if err := tx.Get(&currentVersion, `
		SELECT NT_VERSION
		FROM notification_templates
		WHERE NT_KEY = ?
		FOR UPDATE
	`, key); err != nil {
		return 0, err
	}
	if expectedVersion <= 0 || currentVersion != expectedVersion {
		return 0, ErrNotificationTemplateConflict
	}

	// UPDATED_BY is nullable and the seeded rows carry NULL; an absent
	// administrator is stored the same way rather than as member 0.
	var editor interface{}
	if updatedBy > 0 {
		editor = updatedBy
	}
	if _, err := tx.Exec(`
		UPDATE notification_templates
		SET NT_TITLE = ?, NT_BODY = ?, NT_VERSION = NT_VERSION + 1, UPDATED_AT = NOW(), UPDATED_BY = ?
		WHERE NT_KEY = ?
	`, title, body, editor, key); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return currentVersion + 1, nil
}
