// notification_text_renderer_testing_test.go — One shared template fake for the
// notification senders.
package service

import (
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// FakeNotificationTemplateStore serves optional administrator overrides. It is
// exported so the external service_test package can share it with the
// in-package delivery tests.
type FakeNotificationTemplateStore struct {
	Rows map[string]model.NotificationTemplate
}

func (s *FakeNotificationTemplateStore) ListAll() ([]model.NotificationTemplate, error) {
	return nil, nil
}

func (s *FakeNotificationTemplateStore) Get(key string) (*model.NotificationTemplate, error) {
	row, found := s.Rows[key]
	if !found {
		return nil, nil
	}
	return &row, nil
}

func (s *FakeNotificationTemplateStore) Update(string, string, string, int, int) (int, error) {
	return 0, nil
}

// NewTestNotificationTemplateService wires the real service over the fake store,
// so senders are tested against the production render path. A nil map means no
// administrator has edited anything and every key falls back to the catalog.
func NewTestNotificationTemplateService(rows map[string]model.NotificationTemplate) *NotificationTemplateService {
	return NewNotificationTemplateService(
		&FakeNotificationTemplateStore{Rows: rows},
		cache.New(5*time.Minute, 10*time.Minute),
		zerolog.Nop(),
	)
}
