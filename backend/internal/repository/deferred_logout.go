package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// RevokeMobileSessionByProof binds an authenticated refresh proof to its retained
// original row. A consumed rotation ancestor can end only that account/SID family.
// Nothing is issued or rotated, and expired proof cannot extend this authority.
func (r *AuthRepository) RevokeMobileSessionByProof(ctx context.Context, account int, sid, jti string, proofExpiry time.Time) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var row struct {
		Account       int          `db:"USR_SEQ"`
		SID           string       `db:"MRT_SID"`
		Expiry        time.Time    `db:"EXPIRES_AT"`
		Revoked       sql.NullTime `db:"REVOKED_AT"`
		LegacyRevoked sql.NullTime `db:"MRT_REVOKED_AT"`
	}
	err = tx.GetContext(ctx, &row, `SELECT USR_SEQ,MRT_SID,EXPIRES_AT,REVOKED_AT,MRT_REVOKED_AT FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI=? FOR UPDATE`, jti)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if row.Account != account || row.SID != sid || row.Expiry.Unix() != proofExpiry.Unix() {
		return ErrRefreshTokenInvalid
	}
	if !row.Expiry.After(time.Now()) || row.Revoked.Valid || row.LegacyRevoked.Valid {
		return nil
	}
	if _, err = tx.ExecContext(ctx, `UPDATE ALUMNI_MOBILE_REFRESH_TOKEN SET REVOKED_AT=COALESCE(REVOKED_AT,NOW()),MRT_REVOKED_AT=COALESCE(MRT_REVOKED_AT,NOW()) WHERE USR_SEQ=? AND MRT_SID=? AND REVOKED_AT IS NULL`, account, sid); err != nil {
		return err
	}
	return tx.Commit()
}
