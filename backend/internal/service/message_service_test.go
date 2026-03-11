// message_service_test.go — Unit tests for MessageService SendMessage validation
package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

// mockMessageRepo implements repository.MessageQuerier for testing.
type mockMessageRepo struct {
	insertErr    error
	insertCalled bool
	inbox        []model.Message
	inboxTotal   int
	inboxErr     error
	outbox       []model.Message
	outboxTotal  int
	outboxErr    error
	unreadCount  int
	unreadErr    error
}

func (m *mockMessageRepo) InsertMessage(senderSeq int, recvrSeq int, content string) error {
	m.insertCalled = true
	return m.insertErr
}

func (m *mockMessageRepo) GetInbox(usrSeq int, page int, size int) ([]model.Message, int, error) {
	return m.inbox, m.inboxTotal, m.inboxErr
}

func (m *mockMessageRepo) GetOutbox(usrSeq int, page int, size int) ([]model.Message, int, error) {
	return m.outbox, m.outboxTotal, m.outboxErr
}

func (m *mockMessageRepo) MarkAsRead(amSeq int, usrSeq int) error { return nil }

func (m *mockMessageRepo) DeleteMessage(amSeq int, usrSeq int) error { return nil }

func (m *mockMessageRepo) GetUnreadCount(usrSeq int) (int, error) {
	return m.unreadCount, m.unreadErr
}

// mockProfileRepo implements repository.ProfileQuerier for testing.
type mockProfileRepo struct {
	exists    bool
	existsErr error
}

func (m *mockProfileRepo) CheckUserExists(usrSeq int) (bool, error) {
	return m.exists, m.existsErr
}

func TestSendMessage_EmptyContent(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{exists: true},
	}

	err := svc.SendMessage(1, "Sender", model.SendMessageRequest{
		RecvrSeq: 2,
		Content:  "",
	})
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestSendMessage_ContentTooLong(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{exists: true},
	}

	longContent := strings.Repeat("가", 1001) // 1001 runes
	err := svc.SendMessage(1, "Sender", model.SendMessageRequest{
		RecvrSeq: 2,
		Content:  longContent,
	})
	if err == nil {
		t.Fatal("expected error for content exceeding 1000 characters")
	}
}

func TestSendMessage_SendToSelf(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{exists: true},
	}

	err := svc.SendMessage(5, "Sender", model.SendMessageRequest{
		RecvrSeq: 5,
		Content:  "Hello",
	})
	if err == nil {
		t.Fatal("expected error for sending to self")
	}
}

func TestSendMessage_RecipientNotFound(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{exists: false},
	}

	err := svc.SendMessage(1, "Sender", model.SendMessageRequest{
		RecvrSeq: 999,
		Content:  "Hello",
	})
	if err == nil {
		t.Fatal("expected error for non-existent recipient")
	}
}

func TestSendMessage_RecipientCheckError(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{existsErr: errors.New("db error")},
	}

	err := svc.SendMessage(1, "Sender", model.SendMessageRequest{
		RecvrSeq: 2,
		Content:  "Hello",
	})
	if err == nil {
		t.Fatal("expected error from CheckUserExists failure")
	}
}

func TestSendMessage_Success(t *testing.T) {
	msgRepo := &mockMessageRepo{}
	svc := &MessageService{
		repo:        msgRepo,
		profileRepo: &mockProfileRepo{exists: true},
		notifSvc:    nil,
	}

	err := svc.SendMessage(1, "Sender", model.SendMessageRequest{
		RecvrSeq: 2,
		Content:  "Hello, friend!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !msgRepo.insertCalled {
		t.Error("expected InsertMessage to be called")
	}
}

func TestSendMessage_InsertError(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{insertErr: errors.New("insert failed")},
		profileRepo: &mockProfileRepo{exists: true},
	}

	err := svc.SendMessage(1, "Sender", model.SendMessageRequest{
		RecvrSeq: 2,
		Content:  "Hello",
	})
	if err == nil {
		t.Fatal("expected error from InsertMessage failure")
	}
}
