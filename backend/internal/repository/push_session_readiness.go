package repository

import (
	"errors"
	"github.com/jmoiron/sqlx"
)

// Required even with push sending disabled: deferred logout touches device rows.
func PushSessionSchemaReady(db *sqlx.DB) error {
	var columns int
	err := db.Get(&columns, `SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MOBILE_DEVICE_TOKEN' AND COLUMN_NAME='SESSION_SID' AND DATA_TYPE='char' AND CHARACTER_MAXIMUM_LENGTH=32 AND IS_NULLABLE='YES' AND COLLATION_NAME='ascii_bin'`)
	if err != nil {
		return errors.New("push session schema readiness query failed")
	}
	if columns != 1 {
		return errors.New("push session schema requires migration 085")
	}
	return nil
}
