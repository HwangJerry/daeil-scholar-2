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

// A NULL legacy request must not demote or overwrite a bound registration.
const pushRegistrationSQL = `INSERT INTO ALUMNI_MOBILE_DEVICE_TOKEN
 (USR_SEQ,PLATFORM,DEVICE_TOKEN,LOCALE,APNS_ENVIRONMENT,BUNDLE_ID,SESSION_SID,STATUS,INVALID_COUNT,LAST_SEEN_AT,CREATED_AT,UPDATED_AT)
 VALUES (?,?,?,?,?,?,?,'ACTIVE',0,UTC_TIMESTAMP(),UTC_TIMESTAMP(),UTC_TIMESTAMP())
 ON DUPLICATE KEY UPDATE
 USR_SEQ=IF(VALUES(SESSION_SID) IS NOT NULL OR SESSION_SID IS NULL,VALUES(USR_SEQ),USR_SEQ),
 PLATFORM=IF(VALUES(SESSION_SID) IS NOT NULL OR SESSION_SID IS NULL,VALUES(PLATFORM),PLATFORM),
 LOCALE=IF(VALUES(SESSION_SID) IS NOT NULL OR SESSION_SID IS NULL,VALUES(LOCALE),LOCALE),
 APNS_ENVIRONMENT=IF(VALUES(SESSION_SID) IS NOT NULL OR SESSION_SID IS NULL,VALUES(APNS_ENVIRONMENT),APNS_ENVIRONMENT),
 BUNDLE_ID=IF(VALUES(SESSION_SID) IS NOT NULL OR SESSION_SID IS NULL,VALUES(BUNDLE_ID),BUNDLE_ID),
 STATUS=IF(VALUES(SESSION_SID) IS NOT NULL OR SESSION_SID IS NULL,'ACTIVE',STATUS),
 INVALID_COUNT=IF(VALUES(SESSION_SID) IS NOT NULL OR SESSION_SID IS NULL,0,INVALID_COUNT),
 LAST_SEEN_AT=IF(VALUES(SESSION_SID) IS NOT NULL OR SESSION_SID IS NULL,UTC_TIMESTAMP(),LAST_SEEN_AT),
 UPDATED_AT=IF(VALUES(SESSION_SID) IS NOT NULL OR SESSION_SID IS NULL,UTC_TIMESTAMP(),UPDATED_AT),
 SESSION_SID=COALESCE(VALUES(SESSION_SID),SESSION_SID)`

func (r *PushRepository) RegisterDevice(account int, registration model.PushDeviceRegistration) error {
	if registration.SessionID != "" {
		return r.registerSessionDevice(account, registration)
	}
	_, err := r.db.Exec(pushRegistrationSQL, account, registration.Platform, registration.DeviceToken, registration.Locale, registration.APNSEnvironment, registration.BundleID, nil)
	return err
}

func (r *PushRepository) UnregisterDevice(usrSeq int, deviceToken string) error {
	_, err := r.db.Exec(`
		DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN
		WHERE USR_SEQ = ? AND DEVICE_TOKEN = ? AND SESSION_SID IS NULL
	`, usrSeq, deviceToken)
	return err
}

func (r *PushRepository) ListDevices(usrSeq int) ([]model.PushDeliveryTarget, error) {
	rows, err := r.db.Queryx(`
		SELECT PLATFORM, DEVICE_TOKEN, APNS_ENVIRONMENT, BUNDLE_ID,SESSION_SID
 FROM ALUMNI_MOBILE_DEVICE_TOKEN d
 WHERE USR_SEQ = ? AND STATUS = 'ACTIVE' AND SESSION_SID IS NOT NULL
 AND EXISTS (SELECT 1 FROM ALUMNI_MOBILE_REFRESH_TOKEN r WHERE r.USR_SEQ=d.USR_SEQ AND r.MRT_SID=d.SESSION_SID AND r.CONSUMED_AT IS NULL AND r.REVOKED_AT IS NULL AND r.MRT_REVOKED_AT IS NULL AND r.EXPIRES_AT>NOW())
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
		if err := rows.Scan(&target.Platform, &target.DeviceToken, &environment, &bundleID, &target.SessionID); err != nil {
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
	var noticeEnabled string
	err := r.db.QueryRowx(`
		SELECT MESSAGE_ENABLED, MESSAGE_PREVIEW_ENABLED, NOTICE_ENABLED
		FROM ALUMNI_PUSH_PREFERENCE
		WHERE USR_SEQ = ?
	`, usrSeq).Scan(&messageEnabled, &messagePreviewEnabled, &noticeEnabled)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &model.PushPreferences{
		MessageEnabled:        messageEnabled == "Y",
		MessagePreviewEnabled: messagePreviewEnabled == "Y",
		NoticeEnabled:         noticeEnabled == "Y",
	}, nil
}

// UpsertPreferences writes one preferences PUT in a single statement. A nil
// NoticeEnabled means the client omitted the field, and the COALESCE keeps the
// stored value (or the 'Y' default on insert) without a read-modify-write that
// two concurrent devices could interleave.
func (r *PushRepository) UpsertPreferences(usrSeq int, update model.PushPreferencesUpdate) error {
	noticeEnabled := optionalPushFlag(update.NoticeEnabled)
	_, err := r.db.Exec(`
		INSERT INTO ALUMNI_PUSH_PREFERENCE (
			USR_SEQ,
			MESSAGE_ENABLED,
			MESSAGE_PREVIEW_ENABLED,
			NOTICE_ENABLED,
			CREATED_AT,
			UPDATED_AT
		)
		VALUES (?, ?, ?, COALESCE(?, 'Y'), UTC_TIMESTAMP(), UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE
			MESSAGE_ENABLED = VALUES(MESSAGE_ENABLED),
			MESSAGE_PREVIEW_ENABLED = VALUES(MESSAGE_PREVIEW_ENABLED),
			NOTICE_ENABLED = COALESCE(?, NOTICE_ENABLED),
			UPDATED_AT = UTC_TIMESTAMP()
	`,
		usrSeq,
		pushFlag(update.MessageEnabled),
		pushFlag(update.MessagePreviewEnabled),
		noticeEnabled,
		noticeEnabled,
	)
	return err
}

// optionalPushFlag maps an omitted boolean to SQL NULL, which is what the
// COALESCE in the upsert reads as "leave this column alone".
func optionalPushFlag(enabled *bool) interface{} {
	if enabled == nil {
		return nil
	}
	return pushFlag(*enabled)
}

func pushFlag(enabled bool) string {
	if enabled {
		return "Y"
	}
	return "N"
}

func (r *PushRepository) VerificationStillCurrent(userSeq int, status model.VerificationStatus) (bool, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM ALUMNI_VERIFICATION v
		JOIN WEO_MEMBER m ON m.USR_SEQ = v.USR_SEQ
		WHERE v.USR_SEQ = ? AND v.STATUS = ? AND m.USR_STATUS != 'AAA'`, userSeq, status)
	return count > 0, err
}
