package repository

import (
	"database/sql"
	"fmt"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type IdentityRepository struct {
	DB *sqlx.DB
}

func NewIdentityRepository(db *sqlx.DB) *IdentityRepository {
	return &IdentityRepository{DB: db}
}

func (r *IdentityRepository) UpsertIdentity(identity model.Identity) error {
	normalizedEmail := identity.NormalizedEmail
	var emailValue interface{}
	if normalizedEmail != nil {
		emailValue = *normalizedEmail
	} else {
		emailValue = nil
	}
	_, err := r.DB.Exec(`
		INSERT INTO AUTH_IDENTITY
			(ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL, STATUS, VERIFIED_AT, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, ?, ?, 'ACTIVE', NOW(), NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			SUBJECT_KEY = VALUES(SUBJECT_KEY),
			NORMALIZED_EMAIL = VALUES(NORMALIZED_EMAIL),
			UPDATED_AT = NOW()
	`, identity.AccountSeq, identity.Provider, identity.SubjectKey, emailValue)
	return err
}

func (r *IdentityRepository) FindIdentityByProviderSubject(provider model.IdentityProvider, subjectKey string) (*model.Identity, error) {
	var identity model.Identity
	err := r.DB.Get(&identity, `
		SELECT IDENTITY_ID, ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL, STATUS
		FROM AUTH_IDENTITY
		WHERE PROVIDER = ? AND SUBJECT_KEY = ?
		LIMIT 1
	`, provider, subjectKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find identity by provider/subject: %w", err)
	}
	return &identity, nil
}

func (r *IdentityRepository) FindIdentity(identityID int64) (*model.Identity, error) {
	var identity model.Identity
	err := r.DB.Get(&identity, `
		SELECT IDENTITY_ID, ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL, STATUS
		FROM AUTH_IDENTITY
		WHERE IDENTITY_ID = ?
		LIMIT 1
	`, identityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find identity: %w", err)
	}
	return &identity, nil
}

func (r *IdentityRepository) ListIdentities(accountSeq int) ([]model.Identity, error) {
	var identities []model.Identity
	err := r.DB.Select(&identities, `
		SELECT IDENTITY_ID, ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL, STATUS
		FROM AUTH_IDENTITY
		WHERE ACCOUNT_ID = ?
		ORDER BY IDENTITY_ID ASC
	`, accountSeq)
	if err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}
	return identities, nil
}

func (r *IdentityRepository) DisableIdentity(identityID int64) error {
	_, err := r.DB.Exec(`
		UPDATE AUTH_IDENTITY
		SET STATUS = 'DISABLED', UPDATED_AT = NOW()
		WHERE IDENTITY_ID = ?
	`, identityID)
	return err
}

func (r *IdentityRepository) DeleteIdentity(identityID int64) error {
	_, err := r.DB.Exec(`
		DELETE FROM AUTH_IDENTITY
		WHERE IDENTITY_ID = ?
	`, identityID)
	return err
}
