// app_client_build_repo.go — MariaDB persistence for build numbers observed from mobile requests.
package repository

import (
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// appClientBuildListLimit caps the admin build picker; older builds stay in the
// table but are not useful for choosing a threshold.
const appClientBuildListLimit = 100

type AppClientBuildRepository struct {
	db *sqlx.DB
}

func NewAppClientBuildRepository(db *sqlx.DB) *AppClientBuildRepository {
	return &AppClientBuildRepository{db: db}
}

func (r *AppClientBuildRepository) Upsert(platform string, build int64, versionName string, seenAt time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO app_client_builds (
			PLATFORM, BUILD, VERSION_NAME, FIRST_SEEN_AT, LAST_SEEN_AT
		) VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			VERSION_NAME = VALUES(VERSION_NAME),
			LAST_SEEN_AT = VALUES(LAST_SEEN_AT)
	`, platform, build, versionName, seenAt, seenAt)
	return err
}

func (r *AppClientBuildRepository) List(platform string) ([]model.AppClientBuild, error) {
	builds := make([]model.AppClientBuild, 0)
	err := r.db.Select(&builds, `
		SELECT PLATFORM, BUILD, VERSION_NAME, FIRST_SEEN_AT, LAST_SEEN_AT
		FROM app_client_builds
		WHERE PLATFORM = ?
		ORDER BY BUILD DESC
		LIMIT ?
	`, platform, appClientBuildListLimit)
	return builds, err
}

func (r *AppClientBuildRepository) LatestBuild(platform string) (int64, error) {
	var latest int64
	err := r.db.Get(&latest, `
		SELECT COALESCE(MAX(BUILD), 0)
		FROM app_client_builds
		WHERE PLATFORM = ?
	`, platform)
	return latest, err
}
