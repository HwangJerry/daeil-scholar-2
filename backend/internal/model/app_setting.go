// app_setting.go — Domain model for remotely managed application settings.
package model

import "time"

const (
	AppSettingPublic  = "Y"
	AppSettingPrivate = "N"
)

// AppSetting represents one row in app_settings.
type AppSetting struct {
	Key         string    `db:"AS_KEY" json:"key"`
	Value       string    `db:"AS_VALUE" json:"value"`
	Description string    `db:"AS_DESCRIPTION" json:"description"`
	Public      string    `db:"AS_PUBLIC" json:"public"`
	UpdatedAt   time.Time `db:"UPDATED_AT" json:"updatedAt"`
	UpdatedBy   *int      `db:"UPDATED_BY" json:"updatedBy"`
}
