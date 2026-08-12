package repository

import (
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

var errCredentialOwnerMissing = errors.New("provider credential requires exactly one owner")
var errProviderCredentialInvalid = errors.New("provider credential requires provider and valid nonce")

// CredentialRepository owns AUTH_PASSWORD_CREDENTIAL and AUTH_PROVIDER_CREDENTIAL.
type CredentialRepository struct {
	DB *sqlx.DB
}

func NewCredentialRepository(db *sqlx.DB) *CredentialRepository {
	return &CredentialRepository{DB: db}
}

func (r *CredentialRepository) UpsertPasswordCredential(credential model.PasswordCredential) error {
	if !credential.Provider.SupportsPassword() {
		return errors.New("password credential requires password-supported provider")
	}
	if credential.IdentityID <= 0 {
		return errors.New("password credential requires identity ID")
	}
	if credential.Algorithm == "" {
		return errors.New("password credential requires algorithm")
	}
	if credential.PasswordHash == "" {
		return errors.New("password credential requires hash")
	}

	_, err := r.DB.Exec(`
		INSERT INTO AUTH_PASSWORD_CREDENTIAL
			(IDENTITY_ID, PROVIDER, ALGORITHM, PARAMETERS_TEXT, PASSWORD_HASH, STATUS, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE', NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			ALGORITHM = VALUES(ALGORITHM),
			PARAMETERS_TEXT = VALUES(PARAMETERS_TEXT),
			PASSWORD_HASH = VALUES(PASSWORD_HASH),
			STATUS = 'ACTIVE',
			UPDATED_AT = NOW()
	`, credential.IdentityID, string(credential.Provider), credential.Algorithm, credential.ParametersText, credential.PasswordHash)
	return err
}

func (r *CredentialRepository) FindPasswordCredential(identityID int64) (*model.PasswordCredential, error) {
	var row struct {
		IdentityID   int64          `db:"IDENTITY_ID"`
		Provider     string         `db:"PROVIDER"`
		Algorithm    string         `db:"ALGORITHM"`
		Parameters   sql.NullString `db:"PARAMETERS_TEXT"`
		PasswordHash string         `db:"PASSWORD_HASH"`
		Status       string         `db:"STATUS"`
	}
	err := r.DB.QueryRowx(`
		SELECT IDENTITY_ID, PROVIDER, ALGORITHM, PARAMETERS_TEXT, PASSWORD_HASH, STATUS
		FROM AUTH_PASSWORD_CREDENTIAL
		WHERE IDENTITY_ID = ?
		LIMIT 1
	`, identityID).StructScan(&row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	credential := model.PasswordCredential{
		IdentityID:   row.IdentityID,
		Provider:     model.IdentityProvider(row.Provider),
		Algorithm:    model.PasswordAlgorithm(row.Algorithm),
		PasswordHash: row.PasswordHash,
		Status:       model.PasswordCredentialStatus(row.Status),
	}
	if row.Parameters.Valid {
		credential.ParametersText = &row.Parameters.String
	}
	return &credential, nil
}

func (r *CredentialRepository) RehashPasswordCredential(identityID int64, previousHash string, credential model.PasswordCredential) (bool, error) {
	if credential.IdentityID != identityID || !credential.Provider.SupportsPassword() || credential.Algorithm != model.PasswordAlgorithmArgon2id || credential.PasswordHash == "" {
		return false, errors.New("invalid canonical password rehash")
	}
	result, err := r.DB.Exec(`
		UPDATE AUTH_PASSWORD_CREDENTIAL
		SET ALGORITHM = ?, PARAMETERS_TEXT = ?, PASSWORD_HASH = ?, STATUS = 'ACTIVE', UPDATED_AT = NOW()
		WHERE IDENTITY_ID = ? AND PASSWORD_HASH = ? AND STATUS = 'ACTIVE'
	`, credential.Algorithm, credential.ParametersText, credential.PasswordHash, identityID, previousHash)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected == 1, nil
}

func (r *CredentialRepository) DeletePasswordCredential(identityID int64) error {
	_, err := r.DB.Exec(`
		DELETE FROM AUTH_PASSWORD_CREDENTIAL
		WHERE IDENTITY_ID = ?
	`, identityID)
	return err
}

func (r *CredentialRepository) UpsertProviderCredential(credential model.ProviderCredential) error {
	if !credential.Provider.SupportsProviderCredential() {
		return errors.New("provider credential requires provider-supported provider")
	}
	if len(credential.NonceBytes) != 12 {
		return errProviderCredentialInvalid
	}
	if credential.IdentityID == nil && credential.ContinuationTokenHash == nil {
		return errCredentialOwnerMissing
	}
	if credential.IdentityID != nil && credential.ContinuationTokenHash != nil {
		return errCredentialOwnerMissing
	}
	if credential.IdentityID != nil && *credential.IdentityID <= 0 {
		return errors.New("provider credential requires valid identity ID")
	}
	if credential.KeyID == "" || credential.Algorithm == "" || len(credential.Ciphertext) == 0 {
		return errors.New("provider credential requires key id, algorithm, and ciphertext")
	}

	identityIDParam, continuationParam := r.providerCredentialOwner(credential.IdentityID, credential.ContinuationTokenHash)
	_, err := r.DB.Exec(`
		INSERT INTO AUTH_PROVIDER_CREDENTIAL
			(IDENTITY_ID, CONTINUATION_TOKEN_HASH, PROVIDER, KEY_ID, NONCE_BYTES, ALGORITHM, CIPHERTEXT, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			IDENTITY_ID = VALUES(IDENTITY_ID),
			CONTINUATION_TOKEN_HASH = VALUES(CONTINUATION_TOKEN_HASH),
			KEY_ID = VALUES(KEY_ID),
			NONCE_BYTES = VALUES(NONCE_BYTES),
			ALGORITHM = VALUES(ALGORITHM),
			CIPHERTEXT = VALUES(CIPHERTEXT),
			UPDATED_AT = NOW(),
			REVOKED_AT = NULL
	`, identityIDParam, continuationParam, string(credential.Provider), credential.KeyID, credential.NonceBytes, credential.Algorithm, credential.Ciphertext)
	return err
}

func (r *CredentialRepository) FindProviderCredentialByIdentity(identityID int64, provider model.IdentityProvider) (*model.ProviderCredential, error) {
	return r.findProviderCredential(`
		SELECT CREDENTIAL_ID, IDENTITY_ID, CONTINUATION_TOKEN_HASH, PROVIDER, KEY_ID, NONCE_BYTES, ALGORITHM, CIPHERTEXT
		FROM AUTH_PROVIDER_CREDENTIAL
		WHERE IDENTITY_ID = ? AND PROVIDER = ? AND REVOKED_AT IS NULL
		LIMIT 1
	`, identityID, string(provider))
}

func (r *CredentialRepository) FindProviderCredentialByContinuationToken(continuationTokenHash string) (*model.ProviderCredential, error) {
	return r.findProviderCredential(`
		SELECT CREDENTIAL_ID, IDENTITY_ID, CONTINUATION_TOKEN_HASH, PROVIDER, KEY_ID, NONCE_BYTES, ALGORITHM, CIPHERTEXT
		FROM AUTH_PROVIDER_CREDENTIAL
		WHERE CONTINUATION_TOKEN_HASH = ? AND REVOKED_AT IS NULL
		LIMIT 1
	`, continuationTokenHash)
}

func (r *CredentialRepository) findProviderCredential(query string, args ...interface{}) (*model.ProviderCredential, error) {
	var row struct {
		CredentialID      int64          `db:"CREDENTIAL_ID"`
		IdentityID        sql.NullInt64  `db:"IDENTITY_ID"`
		ContinuationToken sql.NullString `db:"CONTINUATION_TOKEN_HASH"`
		Provider          string         `db:"PROVIDER"`
		KeyID             string         `db:"KEY_ID"`
		NonceBytes        []byte         `db:"NONCE_BYTES"`
		Algorithm         string         `db:"ALGORITHM"`
		Ciphertext        []byte         `db:"CIPHERTEXT"`
	}

	err := r.DB.QueryRowx(query, args...).StructScan(&row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	credential := model.ProviderCredential{
		CredentialID: row.CredentialID,
		Provider:     model.IdentityProvider(row.Provider),
		KeyID:        row.KeyID,
		NonceBytes:   append([]byte(nil), row.NonceBytes...),
		Algorithm:    row.Algorithm,
		Ciphertext:   append([]byte(nil), row.Ciphertext...),
	}
	if row.IdentityID.Valid {
		identityID := row.IdentityID.Int64
		credential.IdentityID = &identityID
	}
	if row.ContinuationToken.Valid {
		continuationToken := row.ContinuationToken.String
		credential.ContinuationTokenHash = &continuationToken
	}
	return &credential, nil
}

func (r *CredentialRepository) DeleteProviderCredential(identityID int64, provider model.IdentityProvider) error {
	_, err := r.DB.Exec(`
		DELETE FROM AUTH_PROVIDER_CREDENTIAL
		WHERE IDENTITY_ID = ? AND PROVIDER = ?
	`, identityID, string(provider))
	return err
}

func (r *CredentialRepository) RevokeProviderCredential(identityID int64, provider model.IdentityProvider) error {
	_, err := r.DB.Exec(`
		UPDATE AUTH_PROVIDER_CREDENTIAL
		SET REVOKED_AT = NOW(), UPDATED_AT = NOW()
		WHERE IDENTITY_ID = ? AND PROVIDER = ?
	`, identityID, string(provider))
	return err
}

func (r *CredentialRepository) providerCredentialOwner(identityID *int64, continuation *string) (any, any) {
	if identityID != nil {
		return *identityID, nil
	}
	if continuation != nil {
		return nil, *continuation
	}
	return nil, nil
}
