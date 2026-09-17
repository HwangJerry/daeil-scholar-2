// phone_verification_repo.go — Database access for SMS phone-ownership verification records.
package repository

import (
	"database/sql"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// PhoneVerificationRepository handles CRUD on the ALUMNI_PHONE_VERIFICATION table.
type PhoneVerificationRepository struct {
	DB *sqlx.DB
}

// NewPhoneVerificationRepository creates a PhoneVerificationRepository backed by the given DB.
func NewPhoneVerificationRepository(db *sqlx.DB) *PhoneVerificationRepository {
	return &PhoneVerificationRepository{DB: db}
}

// InsertVerification records a newly issued code for the given phone number.
func (r *PhoneVerificationRepository) InsertVerification(id, phone, codeHash string, expiresAt time.Time) error {
	_, err := r.DB.Exec(`
		INSERT INTO ALUMNI_PHONE_VERIFICATION
			(APV_ID, PHONE, CODE_HASH, ATTEMPTS, VERIFIED_YN, CONSUMED_YN, EXPIRES_AT, REG_DATE)
		VALUES (?, ?, ?, 0, 'N', 'N', ?, NOW())`,
		id, phone, codeHash, expiresAt,
	)
	return err
}

// CountRecentRequests returns how many codes were issued for a phone number since the given time.
// Used to throttle resends.
func (r *PhoneVerificationRepository) CountRecentRequests(phone string, since time.Time) (int, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION
		WHERE PHONE = ? AND REG_DATE >= ?`,
		phone, since,
	)
	return count, err
}

// FindPendingVerification returns an unverified, unexpired record by its public ID.
// Returns nil when no such record exists.
func (r *PhoneVerificationRepository) FindPendingVerification(id string) (*model.PhoneVerification, error) {
	var record model.PhoneVerification
	err := r.DB.Get(&record, `
		SELECT APV_SEQ, APV_ID, PHONE, CODE_HASH, GRANT_TOKEN_HASH, ATTEMPTS,
		       VERIFIED_YN, CONSUMED_YN, EXPIRES_AT, GRANT_EXPIRES_AT, REG_DATE
		FROM ALUMNI_PHONE_VERIFICATION
		WHERE APV_ID = ? AND VERIFIED_YN = 'N' AND EXPIRES_AT > NOW()`,
		id,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// IncrementAttempts records a failed code entry and returns the resulting attempt count.
func (r *PhoneVerificationRepository) IncrementAttempts(seq int64) (int, error) {
	if _, err := r.DB.Exec(`UPDATE ALUMNI_PHONE_VERIFICATION SET ATTEMPTS = ATTEMPTS + 1 WHERE APV_SEQ = ?`, seq); err != nil {
		return 0, err
	}
	var attempts int
	err := r.DB.Get(&attempts, `SELECT ATTEMPTS FROM ALUMNI_PHONE_VERIFICATION WHERE APV_SEQ = ?`, seq)
	return attempts, err
}

// ExpireVerification invalidates a record immediately, used when attempts run out.
func (r *PhoneVerificationRepository) ExpireVerification(seq int64) error {
	_, err := r.DB.Exec(`UPDATE ALUMNI_PHONE_VERIFICATION SET EXPIRES_AT = NOW() WHERE APV_SEQ = ?`, seq)
	return err
}

// MarkVerified flips the record to verified and attaches the hashed grant token.
func (r *PhoneVerificationRepository) MarkVerified(seq int64, grantTokenHash string, grantExpiresAt time.Time) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_PHONE_VERIFICATION
		SET VERIFIED_YN = 'Y', GRANT_TOKEN_HASH = ?, GRANT_EXPIRES_AT = ?
		WHERE APV_SEQ = ?`,
		grantTokenHash, grantExpiresAt, seq,
	)
	return err
}

// FindUsableGrant returns the phone number a verified, unconsumed, unexpired grant
// was issued for, without spending it. Returns "" when the grant is not usable.
func (r *PhoneVerificationRepository) FindUsableGrant(grantTokenHash string) (string, error) {
	var phone string
	err := r.DB.Get(&phone, `
		SELECT PHONE FROM ALUMNI_PHONE_VERIFICATION
		WHERE GRANT_TOKEN_HASH = ? AND VERIFIED_YN = 'Y' AND CONSUMED_YN = 'N'
		      AND GRANT_EXPIRES_AT > NOW()`,
		grantTokenHash,
	)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return phone, nil
}

// ConsumeGrant atomically marks a verified, unconsumed, unexpired grant as used and
// returns the phone number it was issued for. Returns "" when the grant is not usable.
func (r *PhoneVerificationRepository) ConsumeGrant(grantTokenHash string) (string, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var record struct {
		Seq   int64  `db:"APV_SEQ"`
		Phone string `db:"PHONE"`
	}
	err = tx.Get(&record, `
		SELECT APV_SEQ, PHONE FROM ALUMNI_PHONE_VERIFICATION
		WHERE GRANT_TOKEN_HASH = ? AND VERIFIED_YN = 'Y' AND CONSUMED_YN = 'N'
		      AND GRANT_EXPIRES_AT > NOW()
		FOR UPDATE`,
		grantTokenHash,
	)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	if _, err := tx.Exec(`UPDATE ALUMNI_PHONE_VERIFICATION SET CONSUMED_YN = 'Y' WHERE APV_SEQ = ?`, record.Seq); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return record.Phone, nil
}

// DeleteExpiredBefore removes verification rows older than the cutoff. Codes and
// grant tokens are transient credentials and must not linger past their usefulness.
func (r *PhoneVerificationRepository) DeleteExpiredBefore(cutoff time.Time) (int64, error) {
	result, err := r.DB.Exec(`DELETE FROM ALUMNI_PHONE_VERIFICATION WHERE REG_DATE < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
