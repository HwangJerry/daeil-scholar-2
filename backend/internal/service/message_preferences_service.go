// message_preferences_service.go — Read and save the authenticated member's receiving choice.
package service

import "github.com/dflh-saf/backend/internal/model"

type MessagePreferencesStore interface {
	Get(int) (*model.MessagePreferences, error)
	Save(int, model.MessagePreferences) error
}

type MessagePreferencesService struct{ Store MessagePreferencesStore }

func (s *MessagePreferencesService) Get(usrSeq int) (*model.MessagePreferences, error) {
	return s.Store.Get(usrSeq)
}

func (s *MessagePreferencesService) Save(usrSeq int, preferences model.MessagePreferences) (*model.MessagePreferences, error) {
	if err := s.Store.Save(usrSeq, preferences); err != nil {
		return nil, err
	}
	return &preferences, nil
}
