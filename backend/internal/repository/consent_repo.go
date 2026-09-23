// consent_repo.go — Database access for AUTH_CONSENT, the per-account record of
// accepted notices (privacy, terms, marketing).
package repository

import (
	"time"

	"github.com/jmoiron/sqlx"
)

// ConsentRepository writes consent rows. ACCOUNT_ID is WEO_MEMBER.USR_SEQ.
type ConsentRepository struct {
	DB *sqlx.DB
}

// NewConsentRepository creates a ConsentRepository backed by the given DB.
func NewConsentRepository(db *sqlx.DB) *ConsentRepository {
	return &ConsentRepository{DB: db}
}

// ConsentTableReady reports whether AUTH_CONSENT exists, so deployments that have not
// applied migration 041 can skip recording instead of failing every signup.
func ConsentTableReady(db *sqlx.DB) (bool, error) {
	var count int
	err := db.Get(&count, `
		SELECT COUNT(*) FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'AUTH_CONSENT'`)
	return count == 1, err
}

// RecordAccepted stores an accepted consent for the account. Re-accepting the same
// version (e.g. a retried request) refreshes the acceptance instead of failing on the
// (ACCOUNT_ID, CONSENT_TYPE, CONSENT_VERSION) unique key.
func (r *ConsentRepository) RecordAccepted(accountID int, consentType, version string, required bool, acceptedAt time.Time) error {
	_, err := r.DB.Exec(`
		INSERT INTO AUTH_CONSENT
			(ACCOUNT_ID, CONSENT_TYPE, CONSENT_VERSION, IS_REQUIRED, IS_ACCEPTED, ACCEPTED_AT, WITHDRAWN_AT, CREATED_AT)
		VALUES (?, ?, ?, ?, 1, ?, NULL, ?)
		ON DUPLICATE KEY UPDATE IS_ACCEPTED = 1, ACCEPTED_AT = VALUES(ACCEPTED_AT), WITHDRAWN_AT = NULL`,
		accountID, consentType, version, required, acceptedAt, acceptedAt,
	)
	return err
}
