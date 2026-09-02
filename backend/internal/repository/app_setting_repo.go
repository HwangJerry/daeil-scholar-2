// app_setting_repo.go — MariaDB persistence for application settings.
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AppSettingRepository struct {
	db *sqlx.DB
}

func NewAppSettingRepository(db *sqlx.DB) *AppSettingRepository {
	return &AppSettingRepository{db: db}
}

func (r *AppSettingRepository) ListAll() ([]model.AppSetting, error) {
	settings := make([]model.AppSetting, 0)
	err := r.db.Select(&settings, `
		SELECT AS_KEY, AS_VALUE, AS_DESCRIPTION, AS_PUBLIC, UPDATED_AT, UPDATED_BY
		FROM app_settings
		ORDER BY AS_KEY ASC
	`)
	return settings, err
}

func (r *AppSettingRepository) ListPublic() ([]model.AppSetting, error) {
	settings := make([]model.AppSetting, 0)
	err := r.db.Select(&settings, `
		SELECT AS_KEY, AS_VALUE, AS_DESCRIPTION, AS_PUBLIC, UPDATED_AT, UPDATED_BY
		FROM app_settings
		WHERE AS_PUBLIC = 'Y'
		ORDER BY AS_KEY ASC
	`)
	return settings, err
}

// UpdateValue returns whether the setting exists. MariaDB can report zero
// affected rows when the submitted value and audit fields are unchanged, so a
// zero result is followed by an existence check.
func (r *AppSettingRepository) UpdateValue(key, value string, updatedBy int) (bool, error) {
	result, err := r.db.Exec(`
		UPDATE app_settings
		SET AS_VALUE = ?, UPDATED_AT = NOW(), UPDATED_BY = ?
		WHERE AS_KEY = ?
	`, value, updatedBy, key)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected > 0 {
		return true, nil
	}

	var count int
	if err := r.db.Get(&count, `
		SELECT COUNT(*)
		FROM app_settings
		WHERE AS_KEY = ?
	`, key); err != nil {
		return false, err
	}
	return count > 0, nil
}
