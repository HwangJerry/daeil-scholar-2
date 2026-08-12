// password_reset_repo.go — Database access for password reset tokens and related member lookups
package repository

import (
	"database/sql"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// PasswordResetRepository handles CRUD operations on the ALUMNI_PASSWORD_RESET table
// and password-related member queries.
type PasswordResetRepository struct {
	DB *sqlx.DB
}

// NewPasswordResetRepository creates a PasswordResetRepository backed by the given DB.
func NewPasswordResetRepository(db *sqlx.DB) *PasswordResetRepository {
	return &PasswordResetRepository{DB: db}
}

// InsertToken creates a new password reset token record.
func (r *PasswordResetRepository) InsertToken(usrSeq int, token string, expiresAt time.Time) error {
	_, err := r.DB.Exec(`
		INSERT INTO ALUMNI_PASSWORD_RESET (USR_SEQ, APR_TOKEN, APR_USED_YN, EXPIRES_AT, REG_DATE)
		VALUES (?, ?, 'N', ?, NOW())`,
		usrSeq, token, expiresAt,
	)
	return err
}

// FindValidToken returns an unused, non-expired token record.
// Returns nil if the token is not found, already used, or expired.
func (r *PasswordResetRepository) FindValidToken(token string) (*model.PasswordResetToken, error) {
	var t model.PasswordResetToken
	err := r.DB.Get(&t, `
		SELECT APR_SEQ, USR_SEQ, APR_TOKEN, APR_USED_YN, EXPIRES_AT, REG_DATE
		FROM ALUMNI_PASSWORD_RESET
		WHERE APR_TOKEN = ? AND APR_USED_YN = 'N' AND EXPIRES_AT > NOW()`,
		token,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// MarkTokenUsed sets APR_USED_YN to 'Y' for the given token.
func (r *PasswordResetRepository) MarkTokenUsed(token string) error {
	_, err := r.DB.Exec(`UPDATE ALUMNI_PASSWORD_RESET SET APR_USED_YN = 'Y' WHERE APR_TOKEN = ?`, token)
	return err
}

// UpdatePassword sets a new hashed password for the given member.
func (r *PasswordResetRepository) UpdatePassword(usrSeq int, hashedPwd string) error {
	_, err := r.DB.Exec(`UPDATE WEO_MEMBER SET USR_PWD = ? WHERE USR_SEQ = ?`, hashedPwd, usrSeq)
	return err
}

func (r *PasswordResetRepository) ConfirmResetAtomically(tokenHash, legacyToken, legacyHash string, replacement model.PasswordCredential) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var accountSeq int
	if err := tx.Get(&accountSeq, `
		SELECT USR_SEQ
		FROM ALUMNI_PASSWORD_RESET
		WHERE APR_TOKEN IN (?, ?) AND APR_USED_YN = 'N' AND EXPIRES_AT > NOW()
		LIMIT 1
	`, tokenHash, legacyToken); err != nil {
		if err == sql.ErrNoRows {
			return ErrPasswordResetTokenInvalid
		}
		return err
	}
	var lockedAccountSeq int
	if err := tx.Get(&lockedAccountSeq, `
		SELECT USR_SEQ FROM WEO_MEMBER WHERE USR_SEQ = ? FOR UPDATE
	`, accountSeq); err != nil {
		return err
	}
	var validTokenCount int
	if err := tx.Get(&validTokenCount, `
		SELECT COUNT(*)
		FROM ALUMNI_PASSWORD_RESET
		WHERE USR_SEQ = ? AND APR_TOKEN IN (?, ?)
		  AND APR_USED_YN = 'N' AND EXPIRES_AT > NOW()
	`, accountSeq, tokenHash, legacyToken); err != nil {
		return err
	}
	if validTokenCount == 0 {
		return ErrPasswordResetTokenInvalid
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
	if err := upsertPasswordCredentialsTx(tx, state.activeIdentities, replacement); err != nil {
		return err
	}
	if err := updateLegacyPasswordTx(tx, accountSeq, legacyHash); err != nil {
		return err
	}
	consumeResult, err := tx.Exec(`
		UPDATE ALUMNI_PASSWORD_RESET
		SET APR_USED_YN = 'Y'
		WHERE USR_SEQ = ? AND APR_USED_YN = 'N'
	`, accountSeq)
	if err != nil {
		return err
	}
	consumed, err := consumeResult.RowsAffected()
	if err != nil {
		return err
	}
	if consumed < 1 {
		return ErrPasswordResetTokenInvalid
	}
	if _, err := tx.Exec(`
		UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
		SET REVOKED_AT = COALESCE(REVOKED_AT, NOW()), MRT_REVOKED_AT = COALESCE(MRT_REVOKED_AT, NOW())
		WHERE USR_SEQ = ?
	`, accountSeq); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE AUTH_SESSION_FAMILY
		SET STATUS = 'REVOKED', REVOKED_AT = COALESCE(REVOKED_AT, NOW()),
		    REVOKE_REASON_CODE = COALESCE(REVOKE_REASON_CODE, 'PASSWORD_RESET'), UPDATED_AT = NOW()
		WHERE ACCOUNT_ID = ? AND STATUS = 'ACTIVE'
	`, accountSeq); err != nil {
		return err
	}
	return tx.Commit()
}

// FindPasswordResetAccountByVerifiedEmail uses the canonical verified login
// identity and active account state as password-reset authority.
func (r *PasswordResetRepository) FindPasswordResetAccountByVerifiedEmail(email string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT member.USR_SEQ, member.USR_ID, member.USR_NAME, member.USR_STATUS,
		       member.USR_PHONE, member.USR_FN, member.USR_EMAIL, member.USR_NICK,
		       member.USR_PHOTO, member.REG_DATE
		FROM AUTH_IDENTITY identity
		JOIN AUTH_ACCOUNT_STATE account_state ON account_state.ACCOUNT_ID = identity.ACCOUNT_ID
		JOIN WEO_MEMBER member ON member.USR_SEQ = identity.ACCOUNT_ID
		WHERE identity.PROVIDER = 'EMAIL'
		  AND identity.SUBJECT_KEY = ?
		  AND identity.STATUS = 'ACTIVE'
		  AND identity.VERIFIED_AT IS NOT NULL
		  AND account_state.STATUS = 'ACTIVE'
		LIMIT 1
	`, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindMemberByEmail looks up an active member (USR_STATUS >= 'CCC') by email address.
// Returns nil if no matching member is found.
func (r *PasswordResetRepository) FindMemberByEmail(email string) (*model.User, error) {
	var u model.User
	err := r.DB.Get(&u, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO, REG_DATE
		FROM WEO_MEMBER
		WHERE USR_EMAIL = ? AND USR_STATUS >= 'CCC'
		LIMIT 1`,
		email,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// DeleteExpiredTokens removes all expired token records and returns the count deleted.
func (r *PasswordResetRepository) DeleteExpiredTokens() (int64, error) {
	result, err := r.DB.Exec(`DELETE FROM ALUMNI_PASSWORD_RESET WHERE EXPIRES_AT < NOW()`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetMemberNameBySeq retrieves the member name for the given USR_SEQ.
func (r *PasswordResetRepository) GetMemberNameBySeq(usrSeq int) (string, error) {
	var name string
	err := r.DB.Get(&name, `SELECT USR_NAME FROM WEO_MEMBER WHERE USR_SEQ = ?`, usrSeq)
	if err != nil {
		return "", err
	}
	return name, nil
}
