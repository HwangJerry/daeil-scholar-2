package repository

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrSignupPhoneGrantInvalid = errors.New("signup phone verification grant no longer usable")

// Persist required signup records before committing the member. The conditional
// update also closes the validation-to-commit expiry/concurrent-consumption race.
func persistSignupEvidenceTx(tx *sqlx.Tx, user int, phone string, evidence *model.SignupEvidence) error {
	if evidence == nil {
		return nil
	}
	if evidence.PhoneGrantHash != "" {
		result, err := tx.Exec(`UPDATE ALUMNI_PHONE_VERIFICATION
			SET CONSUMED_YN='Y', CONSUMED_USR_SEQ=?, CONSUMED_AT=NOW()
			WHERE GRANT_TOKEN_HASH=? AND PHONE=? AND VERIFIED_YN='Y'
			AND CONSUMED_YN='N' AND GRANT_EXPIRES_AT>NOW()`, user, evidence.PhoneGrantHash, phone)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return ErrSignupPhoneGrantInvalid
		}
	}
	if evidence.Consent != nil {
		consent := evidence.Consent
		_, err := tx.Exec(`INSERT INTO AUTH_CONSENT
			(ACCOUNT_ID, CONSENT_TYPE, CONSENT_VERSION, IS_REQUIRED, IS_ACCEPTED, ACCEPTED_AT, WITHDRAWN_AT, CREATED_AT)
			VALUES (?, ?, ?, 1, 1, ?, NULL, ?)`, user, model.ConsentTypePrivacy, consent.Version, consent.AcceptedAt, consent.AcceptedAt)
		if err != nil {
			return err
		}
	}
	return nil
}
