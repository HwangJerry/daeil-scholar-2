package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func (r *AuthRepository) RevokeMobileSessionByProof(ctx context.Context, account int, sid, jti string, expiry time.Time) error {
	return r.RevokeMobileSessionByProofWithDevice(ctx, account, sid, jti, expiry, "")
}
func (r *AuthRepository) RevokeMobileSessionByProofWithDevice(ctx context.Context, account int, sid, jti string, expiry time.Time, device string) error {
	_, err := r.RevokeMobileSessionByProofWithDeviceResult(ctx, account, sid, jti, expiry, device)
	return err
}
func (r *AuthRepository) RevokeAllSessionsByProof(ctx context.Context, account int, sid, jti string, expiry time.Time) error {
	_, err := r.revokeSessionsByProof(ctx, account, sid, jti, expiry, "", true)
	return err
}

// Confirmation is granted only after matching retained-proof transaction commits.
func (r *AuthRepository) RevokeMobileSessionByProofWithDeviceResult(ctx context.Context, account int, sid, jti string, expiry time.Time, device string) (bool, error) {
	return r.revokeSessionsByProof(ctx, account, sid, jti, expiry, device, false)
}

// The retained original proof row is locked before session/device mutation.
// Global replay is unauthorized; single-session replay may retry scoped device cleanup.
func (r *AuthRepository) revokeSessionsByProof(ctx context.Context, account int, sid, jti string, expiry time.Time, device string, all bool) (bool, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
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
		if all {
			return false, ErrRefreshTokenInvalid
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if row.Account != account || row.SID != sid || row.Expiry.Unix() != expiry.Unix() {
		return false, ErrRefreshTokenInvalid
	}
	revoked := row.Revoked.Valid || row.LegacyRevoked.Valid
	if !row.Expiry.After(time.Now()) {
		if all {
			return false, ErrRefreshTokenInvalid
		}
		return false, nil
	}
	if all {
		if revoked {
			return false, ErrRefreshTokenInvalid
		}
		if _, err = tx.ExecContext(ctx, `UPDATE ALUMNI_MOBILE_REFRESH_TOKEN SET REVOKED_AT=COALESCE(REVOKED_AT,NOW()),MRT_REVOKED_AT=COALESCE(MRT_REVOKED_AT,NOW()) WHERE USR_SEQ=?`, account); err != nil {
			return false, err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM WEO_MEMBER_LOG WHERE USR_SEQ=?`, account); err != nil {
			return false, err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=?`, account); err != nil {
			return false, err
		}
	} else {
		if !revoked {
			if _, err = tx.ExecContext(ctx, `UPDATE ALUMNI_MOBILE_REFRESH_TOKEN SET REVOKED_AT=COALESCE(REVOKED_AT,NOW()),MRT_REVOKED_AT=COALESCE(MRT_REVOKED_AT,NOW()) WHERE USR_SEQ=? AND MRT_SID=? AND REVOKED_AT IS NULL`, account, sid); err != nil {
				return false, err
			}
		}
		if device != "" {
			if _, err = tx.ExecContext(ctx, `DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=? AND DEVICE_TOKEN=? AND (SESSION_SID=? OR SESSION_SID IS NULL)`, account, device, sid); err != nil {
				return false, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
