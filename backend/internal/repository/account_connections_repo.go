package repository

import (
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrSocialDisconnectPending = errors.New("social disconnect is still being finalized")

// LinkSocialIdentity atomically adds a verified social login method to both
// the legacy connection table and, after the canonical cutover, AUTH_IDENTITY.
func (r *AuthRepository) LinkSocialIdentity(fields SocialAccountFields) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := insertSocialConnectionTx(tx, fields); err != nil {
		return err
	}
	// A DELIVERED entry has already finished credential revocation and local deletion.
	// Remaining claim-release bookkeeping cannot touch a new connection.
	var revocations []struct {
		Status  string `db:"STATUS"`
		Claimed bool   `db:"CLAIMED"`
	}
	if err := tx.Select(&revocations, `SELECT STATUS, (CLAIM_TOKEN IS NOT NULL) AS CLAIMED
        FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX WHERE USR_SEQ=? AND PROVIDER=? AND ACTION='DISCONNECT'
        FOR UPDATE`, fields.USRSeq, fields.Provider); err != nil {
		return err
	}
	for _, revocation := range revocations {
		if revocation.Status != "DELIVERED" {
			return ErrSocialDisconnectPending
		}
	}

	if r.canonicalIdentityReady.Load() {
		if err := relinkSocialIdentityTx(tx, fields); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Explicit, freshly verified linking may reactivate this member's identity.
// Signup insertion deliberately retains its stricter duplicate handling.
func relinkSocialIdentityTx(tx *sqlx.Tx, fields SocialAccountFields) error {
	provider, err := canonicalSocialIdentityProvider(fields.Provider)
	if err != nil {
		return err
	}
	var identity struct {
		ID      int64  `db:"IDENTITY_ID"`
		Account int    `db:"ACCOUNT_ID"`
		Status  string `db:"STATUS"`
	}
	err = tx.Get(&identity, `SELECT IDENTITY_ID, ACCOUNT_ID, STATUS FROM AUTH_IDENTITY
        WHERE PROVIDER=? AND SUBJECT_KEY=? FOR UPDATE`, string(provider), fields.SocialID)
	if errors.Is(err, sql.ErrNoRows) {
		return insertSocialIdentityTx(tx, fields)
	}
	if err != nil {
		return err
	}
	if identity.Account != fields.USRSeq || (identity.Status != "REVOKED" && identity.Status != "ACTIVE") {
		return ErrSocialIdentityAlreadyLinked
	}
	if identity.Status == "ACTIVE" {
		return nil
	}
	_, err = tx.Exec(`UPDATE AUTH_IDENTITY SET STATUS='ACTIVE', REVOKED_AT=NULL,
        VERIFIED_AT=NOW(), UPDATED_AT=NOW() WHERE IDENTITY_ID=? AND ACCOUNT_ID=? AND STATUS='REVOKED'`, identity.ID, fields.USRSeq)
	return err
}

func (r *AuthRepository) GetAccountConnections(usrSeq int) (model.AccountConnections, error) {
	providers := make([]string, 0)
	if err := r.DB.Select(&providers, `
		SELECT NMS_GATE
		FROM WEO_MEMBER_SOCIAL
		WHERE USR_SEQ = ? AND NMS_STATUS IN ('Y', 'ACTIVE')
		ORDER BY NMS_GATE
	`, usrSeq); err != nil {
		return model.AccountConnections{}, err
	}

	var hasPassword bool
	if err := r.DB.Get(&hasPassword, `
		SELECT CASE
			WHEN COALESCE(TRIM(USR_PWD), '') <> '' THEN 1
			ELSE 0
		END
		FROM WEO_MEMBER
		WHERE USR_SEQ = ?
	`, usrSeq); err != nil {
		return model.AccountConnections{}, err
	}

	return model.AccountConnections{
		Providers:   providers,
		HasPassword: hasPassword,
	}, nil
}
