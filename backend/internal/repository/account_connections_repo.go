package repository

import "github.com/dflh-saf/backend/internal/model"

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
	if r.canonicalIdentityReady.Load() {
		if err := insertSocialIdentityTx(tx, fields); err != nil {
			return err
		}
	}
	return tx.Commit()
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
