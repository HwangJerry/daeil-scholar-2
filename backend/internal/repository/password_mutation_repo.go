package repository

import (
	"crypto/subtle"
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrPasswordMismatch           = errors.New("password mismatch")
	ErrPasswordMissing            = errors.New("password missing")
	ErrPasswordIdentityMissing    = errors.New("password identity missing")
	ErrPasswordCredentialDisabled = errors.New("password credential disabled")
	ErrPasswordResetTokenInvalid  = errors.New("password reset token invalid")
)

type PasswordCredentialVerifier func(model.PasswordCredential) (bool, error)

type PasswordMutationRepository struct {
	db *sqlx.DB
}

func NewPasswordMutationRepository(db *sqlx.DB) *PasswordMutationRepository {
	return &PasswordMutationRepository{db: db}
}

func (r *PasswordMutationRepository) ChangePasswordAtomically(
	accountSeq int,
	legacySubmittedHash string,
	legacyReplacementHash string,
	replacement model.PasswordCredential,
	verify PasswordCredentialVerifier,
) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var legacyStoredHash string
	if err := tx.Get(&legacyStoredHash, `
		SELECT IFNULL(USR_PWD, '')
		FROM WEO_MEMBER
		WHERE USR_SEQ = ?
		LIMIT 1
		FOR UPDATE
	`, accountSeq); err != nil {
		return err
	}
	state, err := loadPasswordIdentityStateTx(tx, accountSeq)
	if err != nil {
		return err
	}
	if len(state.activeIdentities) == 0 {
		return ErrPasswordIdentityMissing
	}
	if state.hasDisabledCredential {
		return ErrPasswordCredentialDisabled
	}
	if len(state.activeCredentials) > 0 {
		valid := false
		for _, credential := range state.activeCredentials {
			matches, err := verify(credential)
			if err != nil {
				return err
			}
			valid = valid || matches
		}
		if !valid {
			return ErrPasswordMismatch
		}
	} else {
		if legacyStoredHash == "" {
			return ErrPasswordMissing
		}
		if subtle.ConstantTimeCompare([]byte(legacyStoredHash), []byte(legacySubmittedHash)) != 1 {
			return ErrPasswordMismatch
		}
	}

	if err := upsertPasswordCredentialsTx(tx, state.activeIdentities, replacement); err != nil {
		return err
	}
	if err := updateLegacyPasswordTx(tx, accountSeq, legacyReplacementHash); err != nil {
		return err
	}
	return tx.Commit()
}

type passwordIdentityState struct {
	activeIdentities      []model.Identity
	activeCredentials     []model.PasswordCredential
	hasDisabledCredential bool
}

func loadPasswordIdentityStateTx(tx *sqlx.Tx, accountSeq int) (passwordIdentityState, error) {
	var rows []struct {
		IdentityID       int64          `db:"IDENTITY_ID"`
		Provider         string         `db:"PROVIDER"`
		IdentityStatus   string         `db:"IDENTITY_STATUS"`
		Algorithm        sql.NullString `db:"ALGORITHM"`
		Parameters       sql.NullString `db:"PARAMETERS_TEXT"`
		PasswordHash     sql.NullString `db:"PASSWORD_HASH"`
		CredentialStatus sql.NullString `db:"CREDENTIAL_STATUS"`
	}
	err := tx.Select(&rows, `
		SELECT i.IDENTITY_ID, i.PROVIDER, i.STATUS AS IDENTITY_STATUS,
		       c.ALGORITHM, c.PARAMETERS_TEXT, c.PASSWORD_HASH, c.STATUS AS CREDENTIAL_STATUS
		FROM AUTH_IDENTITY i
		LEFT JOIN AUTH_PASSWORD_CREDENTIAL c ON c.IDENTITY_ID = i.IDENTITY_ID
		WHERE i.ACCOUNT_ID = ? AND i.PROVIDER IN ('EMAIL', 'LOCAL_USERNAME')
		ORDER BY i.IDENTITY_ID
		FOR UPDATE
	`, accountSeq)
	if err != nil {
		return passwordIdentityState{}, err
	}

	state := passwordIdentityState{}
	for _, row := range rows {
		if model.IdentityStatus(row.IdentityStatus) != model.IdentityStatusActive {
			continue
		}
		identity := model.Identity{
			IdentityID: row.IdentityID,
			AccountSeq: accountSeq,
			Provider:   model.IdentityProvider(row.Provider),
			Status:     model.IdentityStatusActive,
		}
		state.activeIdentities = append(state.activeIdentities, identity)
		if !row.CredentialStatus.Valid {
			continue
		}
		if model.PasswordCredentialStatus(row.CredentialStatus.String) != model.PasswordCredentialStatusActive {
			state.hasDisabledCredential = true
			continue
		}
		credential := model.PasswordCredential{
			IdentityID:   row.IdentityID,
			Provider:     model.IdentityProvider(row.Provider),
			Algorithm:    model.PasswordAlgorithm(row.Algorithm.String),
			PasswordHash: row.PasswordHash.String,
			Status:       model.PasswordCredentialStatusActive,
		}
		if row.Parameters.Valid {
			credential.ParametersText = &row.Parameters.String
		}
		state.activeCredentials = append(state.activeCredentials, credential)
	}
	return state, nil
}

func upsertPasswordCredentialsTx(tx *sqlx.Tx, identities []model.Identity, replacement model.PasswordCredential) error {
	for _, identity := range identities {
		if _, err := tx.Exec(`
			INSERT INTO AUTH_PASSWORD_CREDENTIAL
				(IDENTITY_ID, PROVIDER, ALGORITHM, PARAMETERS_TEXT, PASSWORD_HASH, STATUS, CREATED_AT, UPDATED_AT)
			VALUES (?, ?, ?, ?, ?, 'ACTIVE', NOW(), NOW())
			ON DUPLICATE KEY UPDATE
				PROVIDER = VALUES(PROVIDER), ALGORITHM = VALUES(ALGORITHM),
				PARAMETERS_TEXT = VALUES(PARAMETERS_TEXT), PASSWORD_HASH = VALUES(PASSWORD_HASH),
				STATUS = 'ACTIVE', UPDATED_AT = NOW()
		`, identity.IdentityID, string(identity.Provider), replacement.Algorithm, replacement.ParametersText, replacement.PasswordHash); err != nil {
			return err
		}
	}
	return nil
}

func updateLegacyPasswordTx(tx *sqlx.Tx, accountSeq int, legacyHash string) error {
	result, err := tx.Exec(`UPDATE WEO_MEMBER SET USR_PWD = ? WHERE USR_SEQ = ?`, legacyHash, accountSeq)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return sql.ErrNoRows
	}
	return nil
}
