package repository

import (
	"database/sql"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type PushRepository struct {
	db *sqlx.DB
}

func NewPushRepository(db *sqlx.DB) *PushRepository {
	return &PushRepository{db: db}
}

func (r *PushRepository) RegisterDevice(usrSeq int, registration model.PushDeviceRegistration) error {
	_, err := r.db.Exec(`
		INSERT INTO ALUMNI_MOBILE_DEVICE_TOKEN (
			USR_SEQ,
			PLATFORM,
			DEVICE_TOKEN,
			LOCALE,
			APNS_ENVIRONMENT,
			BUNDLE_ID,
			STATUS,
			INVALID_COUNT,
			LAST_SEEN_AT,
			CREATED_AT,
			UPDATED_AT
		)
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', 0, UTC_TIMESTAMP(), UTC_TIMESTAMP(), UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE
			USR_SEQ = VALUES(USR_SEQ),
			PLATFORM = VALUES(PLATFORM),
			LOCALE = VALUES(LOCALE),
			APNS_ENVIRONMENT = VALUES(APNS_ENVIRONMENT),
			BUNDLE_ID = VALUES(BUNDLE_ID),
			STATUS = 'ACTIVE',
			INVALID_COUNT = 0,
			LAST_SEEN_AT = UTC_TIMESTAMP(),
			UPDATED_AT = UTC_TIMESTAMP()
	`,
		usrSeq,
		registration.Platform,
		registration.DeviceToken,
		registration.Locale,
		registration.APNSEnvironment,
		registration.BundleID,
	)
	return err
}

func (r *PushRepository) UnregisterDevice(usrSeq int, deviceToken string) error {
	_, err := r.db.Exec(`
		DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN
		WHERE USR_SEQ = ? AND DEVICE_TOKEN = ?
	`, usrSeq, deviceToken)
	return err
}

func (r *PushRepository) ListDevices(usrSeq int) ([]model.PushDeliveryTarget, error) {
	rows, err := r.db.Queryx(`
		SELECT PLATFORM, DEVICE_TOKEN, APNS_ENVIRONMENT, BUNDLE_ID
		FROM ALUMNI_MOBILE_DEVICE_TOKEN
		WHERE USR_SEQ = ? AND STATUS = 'ACTIVE'
		ORDER BY MDT_SEQ ASC
	`, usrSeq)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	targets := make([]model.PushDeliveryTarget, 0)
	for rows.Next() {
		var target model.PushDeliveryTarget
		var environment sql.NullString
		var bundleID sql.NullString
		if err := rows.Scan(&target.Platform, &target.DeviceToken, &environment, &bundleID); err != nil {
			return nil, err
		}
		if environment.Valid {
			target.APNSEnvironment = environment.String
		}
		if bundleID.Valid {
			target.BundleID = bundleID.String
		}
		targets = append(targets, target)
	}
	return targets, rows.Err()
}

func (r *PushRepository) DeleteDevice(platform, deviceToken string) error {
	_, err := r.db.Exec(`
		DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN
		WHERE PLATFORM = ? AND DEVICE_TOKEN = ?
	`, platform, deviceToken)
	return err
}

func (r *PushRepository) GetPreferences(usrSeq int) (*model.PushPreferences, error) {
	var messageEnabled string
	var messagePreviewEnabled string
	err := r.db.QueryRowx(`
		SELECT MESSAGE_ENABLED, MESSAGE_PREVIEW_ENABLED
		FROM ALUMNI_PUSH_PREFERENCE
		WHERE USR_SEQ = ?
	`, usrSeq).Scan(&messageEnabled, &messagePreviewEnabled)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &model.PushPreferences{
		MessageEnabled:        messageEnabled == "Y",
		MessagePreviewEnabled: messagePreviewEnabled == "Y",
	}, nil
}

func (r *PushRepository) UpsertPreferences(usrSeq int, preferences model.PushPreferences) error {
	_, err := r.db.Exec(`
		INSERT INTO ALUMNI_PUSH_PREFERENCE (
			USR_SEQ,
			MESSAGE_ENABLED,
			MESSAGE_PREVIEW_ENABLED,
			CREATED_AT,
			UPDATED_AT
		)
		VALUES (?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE
			MESSAGE_ENABLED = VALUES(MESSAGE_ENABLED),
			MESSAGE_PREVIEW_ENABLED = VALUES(MESSAGE_PREVIEW_ENABLED),
			UPDATED_AT = UTC_TIMESTAMP()
	`, usrSeq, pushFlag(preferences.MessageEnabled), pushFlag(preferences.MessagePreviewEnabled))
	return err
}

func pushFlag(enabled bool) string {
	if enabled {
		return "Y"
	}
	return "N"
}
