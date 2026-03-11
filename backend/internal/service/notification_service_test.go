// notification_service_test.go — Unit tests for NotificationService GetNotifications
package service

import (
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

// mockNotificationRepo implements repository.NotificationQuerier for testing.
type mockNotificationRepo struct {
	items       []model.Notification
	total       int
	getErr      error
	unreadCount int
	unreadErr   error
}

func (m *mockNotificationRepo) Insert(usrSeq int, notiType, title, body string, refSeq *int) error {
	return nil
}

func (m *mockNotificationRepo) GetByUser(usrSeq, page, size int) ([]model.Notification, int, error) {
	return m.items, m.total, m.getErr
}

func (m *mockNotificationRepo) GetUnreadCount(usrSeq int) (int, error) {
	return m.unreadCount, m.unreadErr
}

func (m *mockNotificationRepo) MarkAsRead(anSeq, usrSeq int) error { return nil }

func (m *mockNotificationRepo) MarkAllAsRead(usrSeq int) error { return nil }

// mockAuthRepoForNotif implements repository.AuthQuerier for testing.
type mockAuthRepoForNotif struct{}

func (m *mockAuthRepoForNotif) GetMemberBySeq(usrSeq int) (*model.User, error) { return nil, nil }

func newTestNotificationService(repo *mockNotificationRepo) *NotificationService {
	emailQueue := make(chan model.EmailMessage, 10)
	logger := zerolog.Nop()
	return &NotificationService{
		repo:        repo,
		authRepo:    &mockAuthRepoForNotif{},
		emailQueue:  emailQueue,
		logger:      logger,
		siteBaseURL: "http://localhost",
	}
}

func TestGetNotifications_SizeClampDefaults(t *testing.T) {
	repo := &mockNotificationRepo{items: []model.Notification{}, total: 0, unreadCount: 0}
	svc := newTestNotificationService(repo)

	resp, err := svc.GetNotifications(1, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Size != 20 {
		t.Errorf("expected size to be clamped to 20, got %d", resp.Size)
	}
	if resp.Page != 1 {
		t.Errorf("expected page to be clamped to 1, got %d", resp.Page)
	}
}

func TestGetNotifications_SizeClampLarge(t *testing.T) {
	repo := &mockNotificationRepo{items: []model.Notification{}, total: 0, unreadCount: 0}
	svc := newTestNotificationService(repo)

	resp, err := svc.GetNotifications(1, 1, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Size != 20 {
		t.Errorf("expected size to be clamped to 20 for value > 50, got %d", resp.Size)
	}
}

func TestGetNotifications_EmptyResult(t *testing.T) {
	repo := &mockNotificationRepo{items: nil, total: 0, unreadCount: 0}
	svc := newTestNotificationService(repo)

	resp, err := svc.GetNotifications(1, 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Items == nil {
		t.Error("expected Items to be non-nil empty slice")
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
	if resp.TotalPages != 0 {
		t.Errorf("expected TotalPages=0, got %d", resp.TotalPages)
	}
}

func TestGetNotifications_Pagination(t *testing.T) {
	items := []model.Notification{
		{ANSeq: 1, ANTitle: "Test1"},
		{ANSeq: 2, ANTitle: "Test2"},
	}
	repo := &mockNotificationRepo{items: items, total: 25, unreadCount: 3}
	svc := newTestNotificationService(repo)

	resp, err := svc.GetNotifications(1, 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCount != 25 {
		t.Errorf("expected TotalCount=25, got %d", resp.TotalCount)
	}
	if resp.TotalPages != 3 {
		t.Errorf("expected TotalPages=3 for 25 items with size 10, got %d", resp.TotalPages)
	}
	if resp.UnreadCount != 3 {
		t.Errorf("expected UnreadCount=3, got %d", resp.UnreadCount)
	}
	if len(resp.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Items))
	}
}

func TestGetNotifications_PageClampNegative(t *testing.T) {
	repo := &mockNotificationRepo{items: []model.Notification{}, total: 0, unreadCount: 0}
	svc := newTestNotificationService(repo)

	resp, err := svc.GetNotifications(1, -5, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Page != 1 {
		t.Errorf("expected page clamped to 1, got %d", resp.Page)
	}
}
