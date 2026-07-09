package repository

import (
	"database/sql"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type PushPreferenceRepository struct {
	DB *sqlx.DB
}

func NewPushPreferenceRepository(db *sqlx.DB) *PushPreferenceRepository {
	return &PushPreferenceRepository{DB: db}
}

func (r *PushPreferenceRepository) GetPreferences(usrSeq int) (model.PushPreferences, error) {
	var row struct {
		NoticeEnabled  string `db:"NOTICE_ENABLED"`
		MessageEnabled string `db:"MESSAGE_ENABLED"`
	}
	err := r.DB.Get(&row, `
		SELECT NOTICE_ENABLED, MESSAGE_ENABLED
		FROM ALUMNI_PUSH_PREFERENCE
		WHERE USR_SEQ = ?
		LIMIT 1
	`, usrSeq)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.DefaultPushPreferences(), nil
		}
		return model.PushPreferences{}, err
	}
	return model.PushPreferences{
		NoticeEnabled:  row.NoticeEnabled == "Y",
		MessageEnabled: row.MessageEnabled == "Y",
	}, nil
}

func (r *PushPreferenceRepository) UpsertPreferences(usrSeq int, preferences model.PushPreferences) (model.PushPreferences, error) {
	_, err := r.DB.Exec(`
		INSERT INTO ALUMNI_PUSH_PREFERENCE
			(USR_SEQ, NOTICE_ENABLED, MESSAGE_ENABLED, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			NOTICE_ENABLED = VALUES(NOTICE_ENABLED),
			MESSAGE_ENABLED = VALUES(MESSAGE_ENABLED),
			UPDATED_AT = NOW()
	`, usrSeq, boolToPreferenceFlag(preferences.NoticeEnabled), boolToPreferenceFlag(preferences.MessageEnabled))
	if err != nil {
		return model.PushPreferences{}, err
	}
	return preferences, nil
}

func boolToPreferenceFlag(enabled bool) string {
	if enabled {
		return "Y"
	}
	return "N"
}
