package repository

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type MobileDeviceTokenRepository struct {
	DB *sqlx.DB
}

type MobileDeviceToken struct {
	UsrSeq      int    `db:"USR_SEQ"`
	DeviceToken string `db:"DEVICE_TOKEN"`
}

func NewMobileDeviceTokenRepository(db *sqlx.DB) *MobileDeviceTokenRepository {
	return &MobileDeviceTokenRepository{DB: db}
}

func (r *MobileDeviceTokenRepository) UpsertToken(usrSeq int, platform string, deviceToken string, locale string) error {
	_, err := r.DB.Exec(`
		INSERT INTO ALUMNI_MOBILE_DEVICE_TOKEN
			(USR_SEQ, PLATFORM, DEVICE_TOKEN, LOCALE, STATUS, LAST_SEEN_AT, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, ?, ?, 'ACTIVE', NOW(), NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			USR_SEQ = VALUES(USR_SEQ),
			PLATFORM = VALUES(PLATFORM),
			LOCALE = VALUES(LOCALE),
			STATUS = 'ACTIVE',
			LAST_SEEN_AT = NOW(),
			UPDATED_AT = NOW()
	`, usrSeq, platform, deviceToken, locale)
	return err
}

func (r *MobileDeviceTokenRepository) DeactivateToken(usrSeq int, deviceToken string) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_MOBILE_DEVICE_TOKEN
		SET STATUS = 'INACTIVE',
		    UPDATED_AT = NOW()
		WHERE USR_SEQ = ? AND DEVICE_TOKEN = ?
	`, usrSeq, deviceToken)
	return err
}

func (r *MobileDeviceTokenRepository) RevokeToken(deviceToken string) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_MOBILE_DEVICE_TOKEN
		SET STATUS = 'REVOKED',
		    INVALID_COUNT = INVALID_COUNT + 1,
		    UPDATED_AT = NOW()
		WHERE DEVICE_TOKEN = ?
	`, deviceToken)
	return err
}

func (r *MobileDeviceTokenRepository) GetActiveTokensByUser(usrSeq int) ([]MobileDeviceToken, error) {
	var tokens []MobileDeviceToken
	err := r.DB.Select(&tokens, `
		SELECT USR_SEQ, DEVICE_TOKEN
		FROM ALUMNI_MOBILE_DEVICE_TOKEN
		WHERE USR_SEQ = ? AND STATUS = 'ACTIVE'
	`, usrSeq)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *MobileDeviceTokenRepository) GetActiveTokensForBroadcast(excludeUsrSeq int) ([]MobileDeviceToken, error) {
	var tokens []MobileDeviceToken
	err := r.DB.Select(&tokens, `
		SELECT dt.USR_SEQ, dt.DEVICE_TOKEN
		FROM ALUMNI_MOBILE_DEVICE_TOKEN dt
		INNER JOIN WEO_MEMBER m ON m.USR_SEQ = dt.USR_SEQ
		WHERE dt.STATUS = 'ACTIVE'
		  AND dt.USR_SEQ <> ?
		  AND m.USR_STATUS IN ('CCC', 'ZZZ')
	`, excludeUsrSeq)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return tokens, nil
}
