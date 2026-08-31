// Identity queries — loads the frozen legacy member and social-link sources.
package main

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func fetchIdentityMembers(ctx context.Context, db *sqlx.DB) ([]identityMemberRow, error) {
	members := make([]identityMemberRow, 0)
	if err := db.SelectContext(ctx, &members, `
		SELECT
			USR_SEQ,
			COALESCE(USR_ID, '') AS USR_ID,
			COALESCE(USR_EMAIL, '') AS USR_EMAIL,
			COALESCE(USR_PWD, '') AS USR_PWD,
			COALESCE(USR_STATUS, '') AS USR_STATUS
		FROM WEO_MEMBER
		ORDER BY USR_SEQ
	`); err != nil {
		return nil, fmt.Errorf("fetch WEO_MEMBER identity source: %w", err)
	}
	return members, nil
}

func fetchIdentitySocialLinks(ctx context.Context, db *sqlx.DB) ([]identitySocialRow, error) {
	links := make([]identitySocialRow, 0)
	if err := db.SelectContext(ctx, &links, `
		SELECT
			USR_SEQ,
			COALESCE(NMS_GATE, '') AS NMS_GATE,
			COALESCE(NMS_ID, '') AS NMS_ID,
			COALESCE(NMS_STATUS, '') AS NMS_STATUS
		FROM WEO_MEMBER_SOCIAL
		ORDER BY USR_SEQ, NMS_GATE, NMS_ID
	`); err != nil {
		return nil, fmt.Errorf("fetch WEO_MEMBER_SOCIAL identity source: %w", err)
	}
	return links, nil
}

func accountIDSet(members []identityMemberRow) map[int]struct{} {
	accountIDs := make(map[int]struct{}, len(members))
	for _, member := range members {
		accountIDs[member.AccountID] = struct{}{}
	}
	return accountIDs
}
