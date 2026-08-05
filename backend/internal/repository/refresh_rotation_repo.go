package repository

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrRefreshTokenInvalid     = errors.New("refresh token is invalid")
	ErrRefreshTokenReplay      = errors.New("refresh token replay detected")
	ErrSessionPrincipalChanged = errors.New("session principal changed before persistence")
)

func (r *AuthRepository) RotateMobileRefreshToken(usrSeq int, sid string, oldJTI string, newJTI string, newExpiresAt time.Time, expectedStatus string) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var row struct {
		SessionID       string       `db:"MRT_SID"`
		ExpiresAt       time.Time    `db:"EXPIRES_AT"`
		ConsumedAt      sql.NullTime `db:"CONSUMED_AT"`
		RevokedAt       sql.NullTime `db:"REVOKED_AT"`
		LegacyRevokedAt sql.NullTime `db:"MRT_REVOKED_AT"`
	}
	err = tx.Get(&row, `
		SELECT MRT_SID, EXPIRES_AT, CONSUMED_AT, REVOKED_AT, MRT_REVOKED_AT
		FROM ALUMNI_MOBILE_REFRESH_TOKEN
		WHERE MRT_JTI = ? AND USR_SEQ = ?
		FOR UPDATE
	`, oldJTI, usrSeq)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRefreshTokenInvalid
	}
	if err != nil {
		return err
	}
	if row.SessionID != sid {
		return ErrRefreshTokenInvalid
	}

	isReplay := row.ConsumedAt.Valid || row.RevokedAt.Valid || row.LegacyRevokedAt.Valid
	isExpired := !row.ExpiresAt.After(time.Now())
	if isReplay || isExpired {
		if _, err := tx.Exec(`
			UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
			SET MRT_REVOKED_AT = COALESCE(MRT_REVOKED_AT, NOW()),
				REVOKED_AT = COALESCE(REVOKED_AT, NOW())
			WHERE USR_SEQ = ? AND MRT_SID = ?
			  AND (MRT_REVOKED_AT IS NULL OR REVOKED_AT IS NULL)
		`, usrSeq, sid); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		if isReplay {
			return ErrRefreshTokenReplay
		}
		return ErrRefreshTokenInvalid
	}

	result, err := tx.Exec(`
		UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
		SET CONSUMED_AT = NOW(), ROTATED_TO_JTI = ?
		WHERE MRT_JTI = ? AND USR_SEQ = ?
		  AND CONSUMED_AT IS NULL AND REVOKED_AT IS NULL AND MRT_REVOKED_AT IS NULL
	`, newJTI, oldJTI, usrSeq)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrRefreshTokenReplay
	}
	insertResult, err := tx.Exec(`
		INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN
			(MRT_JTI, USR_SEQ, MRT_SID, EXPIRES_AT, CREATED_AT)
		SELECT ?, m.USR_SEQ, ?, ?, NOW()
		FROM WEO_MEMBER m
		JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ
		WHERE m.USR_SEQ = ? AND m.USR_STATUS = ?
	`, newJTI, sid, newExpiresAt, usrSeq, expectedStatus)
	if err != nil {
		return err
	}
	inserted, err := insertResult.RowsAffected()
	if err != nil {
		return err
	}
	if inserted != 1 {
		return ErrSessionPrincipalChanged
	}
	return tx.Commit()
}

func (r *AuthRepository) RevokeMobileSession(usrSeq int, sid string) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
		SET MRT_REVOKED_AT = COALESCE(MRT_REVOKED_AT, NOW()),
			REVOKED_AT = COALESCE(REVOKED_AT, NOW())
		WHERE USR_SEQ = ? AND MRT_SID = ?
	`, usrSeq, sid)
	return err
}

func (r *AuthRepository) RevokeAllMobileSessions(usrSeq int) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
		SET MRT_REVOKED_AT = COALESCE(MRT_REVOKED_AT, NOW()),
			REVOKED_AT = COALESCE(REVOKED_AT, NOW())
		WHERE USR_SEQ = ?
	`, usrSeq)
	return err
}

func (r *AuthRepository) IsMobileSessionActive(usrSeq int, sid string) (bool, error) {
	var active bool
	err := r.DB.Get(&active, `
		SELECT EXISTS (
			SELECT 1
			FROM ALUMNI_MOBILE_REFRESH_TOKEN
			WHERE USR_SEQ = ?
			  AND MRT_SID = ?
			  AND CONSUMED_AT IS NULL
			  AND REVOKED_AT IS NULL
			  AND MRT_REVOKED_AT IS NULL
			  AND EXPIRES_AT > NOW()
			LIMIT 1
		) AS ACTIVE
	`, usrSeq, sid)
	return active, err
}
