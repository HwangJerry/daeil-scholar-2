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
	insertErr       error
	insertCalled    bool
	inbox           []model.Message
	inboxTotal      int
	inboxErr        error
	gotInboxPage    int
	gotInboxSize    int
	outbox          []model.Message
	outboxTotal     int
	outboxErr       error
	unreadCount     int
	unreadErr       error
	markSenderSeq   int
	markChanged     bool
	markErr         error
	markConvChanged bool
	markConvErr     error
}

func (m *mockMessageRepo) InsertMessage(senderSeq int, recvrSeq int, content string) error {
	m.insertCalled = true
	return m.insertErr
}

func (m *mockMessageRepo) GetInbox(usrSeq int, page int, size int) ([]model.Message, int, error) {
	m.gotInboxPage = page
	m.gotInboxSize = size
	return m.inbox, m.inboxTotal, m.inboxErr
}

func (m *mockMessageRepo) GetOutbox(usrSeq int, page int, size int) ([]model.Message, int, error) {
	return m.outbox, m.outboxTotal, m.outboxErr
}

func (m *mockMessageRepo) MarkAsRead(amSeq int, usrSeq int) (int, bool, error) {
	return m.markSenderSeq, m.markChanged, m.markErr
}

func (m *mockMessageRepo) DeleteMessage(amSeq int, usrSeq int) error { return nil }

func (m *mockMessageRepo) GetUnreadCount(usrSeq int) (int, error) {
	return m.unreadCount, m.unreadErr
}

func (m *mockMessageRepo) GetConversations(usrSeq int) ([]model.ConversationSummary, error) {
	return nil, nil
}

func (m *mockMessageRepo) GetConversationMessages(usrSeq, otherSeq, page, size int) ([]model.Message, int, error) {
	return nil, 0, nil
}

func (m *mockMessageRepo) MarkConversationRead(usrSeq, senderSeq int) (bool, error) {
	return m.markConvChanged, m.markConvErr
}

type mockMessageNotifier struct {
	readSenderSeq int
	readReaderSeq int
	readCalls     int
}

func (m *mockMessageNotifier) NotifyMessageReceived(int, int, string) {}
func (m *mockMessageNotifier) NotifyMessageSent(int, int)             {}
func (m *mockMessageNotifier) NotifyMessagesRead(senderSeq, readerSeq int) {
	m.readSenderSeq = senderSeq
	m.readReaderSeq = readerSeq
	m.readCalls++
}

// mockProfileRepo implements repository.ProfileQuerier for testing.
type mockProfileRepo struct {
	exists    bool
	existsErr error
}

func (m *mockProfileRepo) CheckUserExists(usrSeq int) (bool, error) {
	return m.exists, m.existsErr
}

// requireErrContains fails the test if err is nil or its message doesn't contain substr.
func requireErrContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Errorf("expected error containing %q, got %q", substr, err.Error())
	}
}

func TestSendMessage_EmptyContent(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{exists: true},
	}
	err := svc.SendMessage(1, "Sender", model.SendMessageRequest{RecvrSeq: 2, Content: ""})
	requireErrContains(t, err, "내용")
}

func TestSendMessage_ContentTooLong(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{exists: true},
	}
	err := svc.SendMessage(1, "Sender", model.SendMessageRequest{
		RecvrSeq: 2,
		Content:  strings.Repeat("가", 1001),
	})
	requireErrContains(t, err, "1000")
}

func TestSendMessage_SendToSelf(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{exists: true},
	}
	err := svc.SendMessage(5, "Sender", model.SendMessageRequest{RecvrSeq: 5, Content: "Hello"})
	requireErrContains(t, err, "자기 자신")
}

func TestSendMessage_RecipientNotFound(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{exists: false},
	}
	err := svc.SendMessage(1, "Sender", model.SendMessageRequest{RecvrSeq: 999, Content: "Hello"})
	requireErrContains(t, err, "존재하지 않는")
}

func TestSendMessage_RecipientCheckError(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{existsErr: errors.New("db error")},
	}
	err := svc.SendMessage(1, "Sender", model.SendMessageRequest{RecvrSeq: 2, Content: "Hello"})
	requireErrContains(t, err, "db error")
}

func TestSendMessage_Success(t *testing.T) {
	msgRepo := &mockMessageRepo{}
	svc := &MessageService{
		repo:        msgRepo,
		profileRepo: &mockProfileRepo{exists: true},
		notifier:    nopMessageNotifier{},
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

// ── GetInbox / GetOutbox pagination tests ─────────────────────────────────────

func TestGetInbox_PaginationClamping(t *testing.T) {
	tests := []struct {
		name          string
		page, size    int
		total         int
		wantPage      int
		wantSize      int
		wantTotalPage int
	}{
		{"defaults for zero", 0, 0, 40, 1, 20, 2},
		{"negative page clamps to 1", -5, 10, 10, 1, 10, 1},
		{"size above cap clamps to 50", 1, 100, 55, 1, 50, 2},
		{"normal values pass through", 3, 15, 45, 3, 15, 3},
		{"zero total gives zero pages", 1, 20, 0, 1, 20, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &MessageService{
				repo: &mockMessageRepo{
					inbox:      []model.Message{},
					inboxTotal: tc.total,
				},
				profileRepo: &mockProfileRepo{},
			}
			resp, err := svc.GetInbox(1, tc.page, tc.size)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Page != tc.wantPage {
				t.Errorf("page: got %d, want %d", resp.Page, tc.wantPage)
			}
			if resp.Size != tc.wantSize {
				t.Errorf("size: got %d, want %d", resp.Size, tc.wantSize)
			}
			if resp.TotalPages != tc.wantTotalPage {
				t.Errorf("totalPages: got %d, want %d", resp.TotalPages, tc.wantTotalPage)
			}
		})
	}
}

func TestGetInbox_NilMessagesBecomesEmptySlice(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{inbox: nil, inboxTotal: 0},
		profileRepo: &mockProfileRepo{},
	}
	resp, err := svc.GetInbox(1, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Items == nil {
		t.Error("expected non-nil Items slice, got nil")
	}
}

func TestGetInbox_RepoError(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{inboxErr: errors.New("db down")},
		profileRepo: &mockProfileRepo{},
	}
	_, err := svc.GetInbox(1, 1, 20)
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

func TestGetOutbox_PaginationClamping(t *testing.T) {
	svc := &MessageService{
		repo: &mockMessageRepo{
			outbox:      []model.Message{},
			outboxTotal: 30,
		},
		profileRepo: &mockProfileRepo{},
	}
	// size=0 should clamp to 20, producing 2 total pages for 30 items
	resp, err := svc.GetOutbox(1, 1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Size != 20 {
		t.Errorf("expected size 20, got %d", resp.Size)
	}
	if resp.TotalPages != 2 {
		t.Errorf("expected 2 total pages, got %d", resp.TotalPages)
	}
}

// TestGetInbox_ClampsArgsPassedToRepo verifies that the service actually passes
// the clamped page/size values to the repo, not the raw inputs.
func TestGetInbox_ClampsArgsPassedToRepo(t *testing.T) {
	mock := &mockMessageRepo{}
	svc := &MessageService{repo: mock, profileRepo: &mockProfileRepo{}}

	// page=0 → clamp to 1; size=200 → clamp to 50
	_, err := svc.GetInbox(1, 0, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.gotInboxPage != 1 {
		t.Errorf("expected page=1 passed to repo after clamping, got %d", mock.gotInboxPage)
	}
	if mock.gotInboxSize != 50 {
		t.Errorf("expected size=50 passed to repo after clamping, got %d", mock.gotInboxSize)
	}
}

// ── GetConversations tests ────────────────────────────────────────────────────

func TestGetConversations_NilBecomesEmptySlice(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{},
	}
	resp, err := svc.GetConversations(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Items == nil {
		t.Error("expected non-nil Items slice, got nil")
	}
}

func TestMarkAsRead_NotifiesSenderWhenChanged(t *testing.T) {
	notifier := &mockMessageNotifier{}
	svc := &MessageService{
		repo:        &mockMessageRepo{markSenderSeq: 7, markChanged: true},
		profileRepo: &mockProfileRepo{},
		notifier:    notifier,
	}

	if err := svc.MarkAsRead(100, 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notifier.readCalls != 1 {
		t.Fatalf("expected one read notification, got %d", notifier.readCalls)
	}
	if notifier.readSenderSeq != 7 || notifier.readReaderSeq != 3 {
		t.Fatalf("unexpected notification payload: sender=%d reader=%d", notifier.readSenderSeq, notifier.readReaderSeq)
	}
}

func TestMarkAsRead_SkipsNotificationWhenUnchanged(t *testing.T) {
	notifier := &mockMessageNotifier{}
	svc := &MessageService{
		repo:        &mockMessageRepo{markSenderSeq: 7, markChanged: false},
		profileRepo: &mockProfileRepo{},
		notifier:    notifier,
	}

	if err := svc.MarkAsRead(100, 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notifier.readCalls != 0 {
		t.Fatalf("expected no read notification, got %d", notifier.readCalls)
	}
}

func TestMarkConversationRead_NotifiesSenderWhenChanged(t *testing.T) {
	notifier := &mockMessageNotifier{}
	svc := &MessageService{
		repo:        &mockMessageRepo{markConvChanged: true},
		profileRepo: &mockProfileRepo{},
		notifier:    notifier,
	}

	if err := svc.MarkConversationRead(3, 7); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notifier.readCalls != 1 {
		t.Fatalf("expected one read notification, got %d", notifier.readCalls)
	}
	if notifier.readSenderSeq != 7 || notifier.readReaderSeq != 3 {
		t.Fatalf("unexpected notification payload: sender=%d reader=%d", notifier.readSenderSeq, notifier.readReaderSeq)
	}
}
