package repository

import (
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
)

func (r *PushRepository) UnregisterDeviceForSession(account int, sid, device string) error {
	_, err := r.db.Exec(`DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=? AND DEVICE_TOKEN=? AND (SESSION_SID=? OR SESSION_SID IS NULL)`, account, device, sid)
	return err
}

// The live refresh family is locked before the device row. Revocation follows
// this same order, so an auth-inflight registration cannot resurrect ended SID.
func (r *PushRepository) registerSessionDevice(account int, registration model.PushDeviceRegistration) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Locate a candidate without range locks, then make the first locking read
	// its exact PK. A rotation between the reads is rejected without any write.
	var proof string
	err = tx.Get(&proof, `SELECT MRT_JTI FROM ALUMNI_MOBILE_REFRESH_TOKEN FORCE INDEX (IDX_MRT_SID) WHERE USR_SEQ=? AND MRT_SID=? AND CONSUMED_AT IS NULL AND REVOKED_AT IS NULL AND MRT_REVOKED_AT IS NULL AND EXPIRES_AT>NOW() ORDER BY MRT_JTI LIMIT 1`, account, registration.SessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRefreshTokenInvalid
	}
	if err != nil {
		return err
	}
	var live int
	err = tx.Get(&live, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI=? AND USR_SEQ=? AND MRT_SID=? AND CONSUMED_AT IS NULL AND REVOKED_AT IS NULL AND MRT_REVOKED_AT IS NULL AND EXPIRES_AT>NOW() FOR UPDATE`, proof, account, registration.SessionID)
	if err != nil {
		return err
	}
	if live != 1 {
		return ErrRefreshTokenInvalid
	}
	if _, err = tx.Exec(pushRegistrationSQL, account, registration.Platform, registration.DeviceToken, registration.Locale, registration.APNSEnvironment, registration.BundleID, registration.SessionID); err != nil {
		return err
	}
	return tx.Commit()
}

// Bound ownership and a live family must still match before each delivery.
func (r *PushRepository) DeliveryTargetStillCurrent(account int, target model.PushDeliveryTarget) (bool, error) {
	if target.SessionID == "" {
		return false, nil
	}
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN d WHERE d.USR_SEQ=? AND d.PLATFORM=? AND d.DEVICE_TOKEN=? AND d.SESSION_SID=? AND d.STATUS='ACTIVE' AND EXISTS (SELECT 1 FROM ALUMNI_MOBILE_REFRESH_TOKEN r WHERE r.USR_SEQ=d.USR_SEQ AND r.MRT_SID=d.SESSION_SID AND r.CONSUMED_AT IS NULL AND r.REVOKED_AT IS NULL AND r.MRT_REVOKED_AT IS NULL AND r.EXPIRES_AT>NOW())`, account, target.Platform, target.DeviceToken, target.SessionID)
	return count == 1, err
}

func (r *PushRepository) DeleteDeliveryTarget(account int, target model.PushDeliveryTarget) error {
	if target.SessionID == "" {
		return nil
	}
	_, err := r.db.Exec(`DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=? AND PLATFORM=? AND DEVICE_TOKEN=? AND SESSION_SID=?`, account, target.Platform, target.DeviceToken, target.SessionID)
	return err
}
