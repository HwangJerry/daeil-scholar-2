// message_preferences_repo.go — Durable receiving preference on the member record.
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type MessagePreferencesRepository struct{ DB *sqlx.DB }

func (r *MessagePreferencesRepository) Get(usrSeq int) (*model.MessagePreferences, error) {
	var allowed bool
	err := r.DB.Get(&allowed, `SELECT USR_MESSAGE_ALLOWED = 'Y' FROM WEO_MEMBER WHERE USR_SEQ = ?`, usrSeq)
	if err != nil {
		return nil, err
	}
	return &model.MessagePreferences{MessageAllowed: allowed}, nil
}

func (r *MessagePreferencesRepository) Save(usrSeq int, preferences model.MessagePreferences) error {
	flag := "N"
	if preferences.MessageAllowed {
		flag = "Y"
	}
	_, err := r.DB.Exec(`UPDATE WEO_MEMBER SET USR_MESSAGE_ALLOWED = ? WHERE USR_SEQ = ?`, flag, usrSeq)
	return err
}

func (r *MessageRepository) CanReceiveMessages(usrSeq int) (bool, error) {
	preferences, err := (&MessagePreferencesRepository{DB: r.DB}).Get(usrSeq)
	if err != nil {
		return false, err
	}
	return preferences.MessageAllowed, nil
}
