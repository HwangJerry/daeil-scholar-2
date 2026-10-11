package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
)

// SocialIdentityOwnership contains only the lifecycle facts needed to decide
// whether verified provider authentication may start signup. No owner identifier
// or provider credential needs to leave the repository for this decision.
type SocialIdentityOwnership struct {
	IdentityStatus string `db:"IDENTITY_STATUS"`
	MemberStatus   string `db:"USR_STATUS"`
}

func (r *AuthRepository) FindSocialIdentityOwnership(ctx context.Context, provider model.SocialProvider, subject string) (*SocialIdentityOwnership, error) {
	// Preserve the legacy-only rollout until canonical identities are authoritative.
	if !r.canonicalIdentityReady.Load() {
		return nil, nil
	}
	canonical, err := canonicalSocialIdentityProvider(string(provider))
	if err != nil {
		return nil, err
	}
	var ownership SocialIdentityOwnership
	err = r.DB.GetContext(ctx, &ownership, `
		SELECT i.STATUS AS IDENTITY_STATUS, COALESCE(m.USR_STATUS, '') AS USR_STATUS
		FROM AUTH_IDENTITY i
		LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = i.ACCOUNT_ID
		WHERE i.PROVIDER = ? AND i.SUBJECT_KEY = ?
		LIMIT 1
	`, string(canonical), subject)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ownership, nil
}
