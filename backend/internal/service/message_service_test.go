// message_service_test.go — Unit tests for MessageService SendMessage validation
package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

// mockMessageRepo implements repository.MessageQuerier for testing.
type mockMessageRepo struct {
	insertErr            error
	insertCalled         bool
	findResult           *model.SendMessageResponse
	findErr              error
	acceptResult         *model.SendMessageResponse
	approved             bool
	approvedSet          bool
	approvedErr          error
	inbox                []model.Message
	inboxTotal           int
	inboxErr             error
	gotInboxPage         int
	gotInboxSize         int
	outbox               []model.Message
	outboxTotal          int
	outboxErr            error
	unreadCount          int
	unreadErr            error
	markSenderSeq        int
	markChanged          bool
	markErr              error
	markConvChanged      bool
	markConvErr          error
	markConvThrough      int64
	conversations        []model.ConversationSummary
	conversationsErr     error
	conversationBefore   *time.Time
	conversationBeforeID int64
	conversationLimit    int
	canonicalMessages    []model.Message
	canonicalMessagesErr error
	canonicalBefore      *time.Time
	canonicalBeforeID    int64
	canonicalLimit       int
}

func (m *mockMessageRepo) FindAcceptedMessage(int, string) (*model.SendMessageResponse, error) {
	return m.findResult, m.findErr
}

func (m *mockMessageRepo) AcceptMessage(senderSeq, recvrSeq int, clientMessageID, content string) (*model.SendMessageResponse, error) {
	m.insertCalled = true
	if m.acceptResult == nil {
		m.acceptResult = &model.SendMessageResponse{
			MessageID:       9001,
			ClientMessageID: clientMessageID,
			Status:          "accepted",
			CreatedAt:       "2026-07-28T01:00:00Z",
			WasCreated:      true,
		}
	}
	return m.acceptResult, m.insertErr
}

func (m *mockMessageRepo) IsApprovedAlumni(int) (bool, error) {
	if !m.approvedSet {
		return true, m.approvedErr
	}
	return m.approved, m.approvedErr
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

func (m *mockMessageRepo) GetConversations(usrSeq int, beforeCreatedAt *time.Time, beforeMessageID int64, limit int) ([]model.ConversationSummary, error) {
	m.conversationBefore = beforeCreatedAt
	m.conversationBeforeID = beforeMessageID
	m.conversationLimit = limit
	return m.conversations, m.conversationsErr
}

func (m *mockMessageRepo) GetConversationMessages(usrSeq, otherSeq, page, size int) ([]model.Message, int, error) {
	return nil, 0, nil
}

func (m *mockMessageRepo) GetCanonicalConversationMessages(usrSeq, otherSeq int, beforeCreatedAt *time.Time, beforeMessageID int64, limit int) ([]model.Message, error) {
	m.canonicalBefore = beforeCreatedAt
	m.canonicalBeforeID = beforeMessageID
	m.canonicalLimit = limit
	return m.canonicalMessages, m.canonicalMessagesErr
}

func (m *mockMessageRepo) MarkConversationRead(usrSeq, senderSeq int, throughMessageID int64) (bool, error) {
	m.markConvThrough = throughMessageID
	return m.markConvChanged, m.markConvErr
}

type mockMessageNotifier struct {
	readSenderSeq int
	readReaderSeq int
	readThroughID int64
	readAt        string
	readCalls     int
	receivedCalls int
	sentCalls     int
}

func (m *mockMessageNotifier) NotifyMessageReceived(int, int, string, *model.SendMessageResponse, string) {
	m.receivedCalls++
}
func (m *mockMessageNotifier) NotifyMessageSent(int, int) { m.sentCalls++ }
func (m *mockMessageNotifier) NotifyMessagesRead(senderSeq, readerSeq int, throughMessageID int64, readAt string) {
	m.readSenderSeq = senderSeq
	m.readReaderSeq = readerSeq
	m.readThroughID = throughMessageID
	m.readAt = readAt
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
	_, err := svc.SendMessage(1, "Sender", model.SendMessageRequest{RecvrSeq: 2, Content: ""})
	requireErrContains(t, err, "내용")
}

func TestSendMessage_ContentTooLong(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{},
		profileRepo: &mockProfileRepo{exists: true},
	}
	_, err := svc.SendMessage(1, "Sender", model.SendMessageRequest{
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
	_, err := svc.SendMessage(5, "Sender", model.SendMessageRequest{RecvrSeq: 5, ClientMessageID: "self", Content: "Hello"})
	requireErrContains(t, err, "자기 자신")
}

func TestSendMessage_RecipientNotFound(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{approvedSet: true, approved: false},
		profileRepo: &mockProfileRepo{exists: false},
	}
	_, err := svc.SendMessage(1, "Sender", model.SendMessageRequest{RecvrSeq: 999, ClientMessageID: "missing", Content: "Hello"})
	requireErrContains(t, err, "승인된 동문")
}

func TestSendMessage_RecipientCheckError(t *testing.T) {
	svc := &MessageService{
		repo:        &mockMessageRepo{approvedErr: errors.New("db error")},
		profileRepo: &mockProfileRepo{existsErr: errors.New("db error")},
	}
	_, err := svc.SendMessage(1, "Sender", model.SendMessageRequest{RecvrSeq: 2, ClientMessageID: "check-error", Content: "Hello"})
	requireErrContains(t, err, "db error")
}

func TestSendMessage_Success(t *testing.T) {
	msgRepo := &mockMessageRepo{}
	svc := &MessageService{
		repo:        msgRepo,
		profileRepo: &mockProfileRepo{exists: true},
		notifier:    nopMessageNotifier{},
	}

	_, err := svc.SendMessage(1, "Sender", model.SendMessageRequest{
		RecvrSeq:        2,
		ClientMessageID: "success",
		Content:         "Hello, friend!",
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

	_, err := svc.SendMessage(1, "Sender", model.SendMessageRequest{
		RecvrSeq:        2,
		ClientMessageID: "insert-error",
		Content:         "Hello",
	})
	if err == nil {
		t.Fatal("expected error from InsertMessage failure")
	}
}

func TestSendMessage_IdempotentReplayReturnsOriginalAfterRecipientStateChanges(t *testing.T) {
	original := &model.SendMessageResponse{
		MessageID:       9001,
		ClientMessageID: "replay-key",
		Status:          "accepted",
		CreatedAt:       "2026-07-28T01:00:00Z",
		WasCreated:      false,
	}
	svc := &MessageService{
		repo: &mockMessageRepo{
			findResult:  original,
			approvedSet: true,
			approved:    false,
		},
		profileRepo: &mockProfileRepo{},
		notifier:    nopMessageNotifier{},
	}

	accepted, err := svc.SendMessage(1, "Sender", model.SendMessageRequest{
		UserSeq:         2,
		ClientMessageID: "replay-key",
		Content:         "original content",
	})
	if err != nil {
		t.Fatalf("replay returned error: %v", err)
	}
	if accepted.MessageID != original.MessageID || accepted.CreatedAt != original.CreatedAt {
		t.Fatalf("accepted = %+v, want original %+v", accepted, original)
	}
}

func TestSendMessage_RecipientBlockedAcceptsWithoutDeliveryNotification(t *testing.T) {
	notifier := &mockMessageNotifier{}
	svc := &MessageService{
		repo: &mockMessageRepo{acceptResult: &model.SendMessageResponse{
			MessageID:          9002,
			ClientMessageID:    "blocked-key",
			Status:             "accepted",
			CreatedAt:          "2026-07-28T01:00:00Z",
			WasCreated:         true,
			VisibleToRecipient: "N",
		}},
		profileRepo: &mockProfileRepo{},
		notifier:    notifier,
	}

	accepted, err := svc.SendMessage(101, "Sender", model.SendMessageRequest{
		UserSeq:         202,
		ClientMessageID: "blocked-key",
		Content:         "안녕하세요.",
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if accepted.Status != "accepted" {
		t.Fatalf("status = %q, want accepted", accepted.Status)
	}
	if notifier.receivedCalls != 0 || notifier.sentCalls != 0 {
		t.Fatalf("notifier calls = received:%d sent:%d, want zero", notifier.receivedCalls, notifier.sentCalls)
	}
}

func TestSendMessage_DeliverableFirstAcceptanceNotifiesOnce(t *testing.T) {
	notifier := &mockMessageNotifier{}
	svc := &MessageService{
		repo: &mockMessageRepo{acceptResult: &model.SendMessageResponse{
			MessageID:          9003,
			ClientMessageID:    "deliverable-key",
			Status:             "accepted",
			CreatedAt:          "2026-07-28T01:00:00Z",
			WasCreated:         true,
			VisibleToRecipient: "Y",
		}},
		profileRepo: &mockProfileRepo{},
		notifier:    notifier,
	}

	_, err := svc.SendMessage(101, "Sender", model.SendMessageRequest{
		UserSeq:         202,
		ClientMessageID: "deliverable-key",
		Content:         "안녕하세요.",
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if notifier.receivedCalls != 1 || notifier.sentCalls != 1 {
		t.Fatalf("notifier calls = received:%d sent:%d, want one each", notifier.receivedCalls, notifier.sentCalls)
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
	resp, err := svc.GetConversations(1, "", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Items == nil {
		t.Error("expected non-nil Items slice, got nil")
	}
}

func TestGetConversations_ReturnsOpaqueNextCursorFromSizePlusOne(t *testing.T) {
	newest := time.Date(2026, 7, 28, 1, 0, 0, 0, time.UTC)
	repo := &mockMessageRepo{conversations: []model.ConversationSummary{
		{UserSeq: 2, CursorCreatedAt: newest, CursorLastMessageID: 9002},
		{UserSeq: 3, CursorCreatedAt: newest.Add(-time.Hour), CursorLastMessageID: 9001},
	}}
	svc := &MessageService{
		repo:        repo,
		profileRepo: &mockProfileRepo{},
	}

	resp, err := svc.GetConversations(1, "", 1)
	if err != nil {
		t.Fatalf("GetConversations: %v", err)
	}
	if len(resp.Items) != 1 || !resp.HasMore || resp.NextCursor == nil || *resp.NextCursor == "" {
		t.Fatalf("response = %+v, want one item and opaque next cursor", resp)
	}
	if strings.Contains(*resp.NextCursor, "2026") || strings.Contains(*resp.NextCursor, "9002") {
		t.Fatalf("cursor must be opaque, got %q", *resp.NextCursor)
	}
	if repo.conversationLimit != 2 {
		t.Fatalf("repository limit = %d, want size+1 = 2", repo.conversationLimit)
	}
}

func TestGetConversations_RejectsMalformedCursor(t *testing.T) {
	svc := &MessageService{repo: &mockMessageRepo{}, profileRepo: &mockProfileRepo{}}

	_, err := svc.GetConversations(1, "not-an-opaque-cursor", 20)
	var validationErr *model.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
}

func TestGetConversations_ContinuesStrictlyBeforeStableCursor(t *testing.T) {
	newest := time.Date(2026, 7, 28, 1, 0, 0, 0, time.UTC)
	cursor, err := encodeMessageCursor(newest, 9002)
	if err != nil {
		t.Fatalf("encodeMessageCursor: %v", err)
	}
	repo := &mockMessageRepo{conversations: []model.ConversationSummary{
		{UserSeq: 2, CursorCreatedAt: newest, CursorLastMessageID: 9002},
		{UserSeq: 3, CursorCreatedAt: newest.Add(-time.Hour), CursorLastMessageID: 9001},
		{UserSeq: 4, CursorCreatedAt: newest.Add(-2 * time.Hour), CursorLastMessageID: 9000},
	}}
	svc := &MessageService{
		repo:        repo,
		profileRepo: &mockProfileRepo{},
	}

	resp, err := svc.GetConversations(1, cursor, 1)
	if err != nil {
		t.Fatalf("GetConversations: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].UserSeq != 3 {
		t.Fatalf("items = %+v, want first item strictly older than cursor", resp.Items)
	}
	if repo.conversationBefore == nil || !repo.conversationBefore.Equal(newest) || repo.conversationBeforeID != 9002 {
		t.Fatalf("repository cursor = (%v, %d), want (%v, 9002)", repo.conversationBefore, repo.conversationBeforeID, newest)
	}
}

func TestGetConversationMessages_ReturnsLatestPageWithOpaqueNextCursor(t *testing.T) {
	repo := &mockMessageRepo{canonicalMessages: []model.Message{
		{AMSeq: 9003, RegDate: "2026-07-28T01:00:00Z"},
		{AMSeq: 9002, RegDate: "2026-07-28T01:00:00Z"},
		{AMSeq: 9001, RegDate: "2026-07-28T00:00:00Z"},
	}}
	svc := &MessageService{repo: repo, profileRepo: &mockProfileRepo{}}

	resp, err := svc.GetConversationMessages(101, 202, "", 1)
	if err != nil {
		t.Fatalf("GetConversationMessages: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].MessageID != 9003 {
		t.Fatalf("items = %+v, want latest message only", resp.Items)
	}
	if !resp.HasMore || resp.NextCursor == nil || *resp.NextCursor == "" {
		t.Fatalf("pagination = %+v, want opaque next cursor", resp)
	}
	if strings.Contains(*resp.NextCursor, "9003") || strings.Contains(*resp.NextCursor, "2026") {
		t.Fatalf("nextCursor must be opaque, got %q", *resp.NextCursor)
	}
	if repo.canonicalLimit != 2 {
		t.Fatalf("repository limit = %d, want size+1 = 2", repo.canonicalLimit)
	}
}

func TestGetConversationMessages_ContinuesStrictlyBeforeStableCursor(t *testing.T) {
	createdAt := time.Date(2026, 7, 28, 1, 0, 0, 0, time.UTC)
	before, err := encodeMessageCursor(createdAt, 9003)
	if err != nil {
		t.Fatalf("encodeMessageCursor: %v", err)
	}
	repo := &mockMessageRepo{canonicalMessages: []model.Message{
		{AMSeq: 9003, RegDate: "2026-07-28T01:00:00Z"},
		{AMSeq: 9002, RegDate: "2026-07-28T01:00:00Z"},
		{AMSeq: 9001, RegDate: "2026-07-28T00:00:00Z"},
	}}
	svc := &MessageService{repo: repo, profileRepo: &mockProfileRepo{}}

	resp, err := svc.GetConversationMessages(101, 202, before, 1)
	if err != nil {
		t.Fatalf("GetConversationMessages: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].MessageID != 9002 {
		t.Fatalf("items = %+v, want same-timestamp lower messageId", resp.Items)
	}
	if repo.canonicalBefore == nil || !repo.canonicalBefore.Equal(createdAt) || repo.canonicalBeforeID != 9003 {
		t.Fatalf("repository cursor = (%v, %d), want (%v, 9003)", repo.canonicalBefore, repo.canonicalBeforeID, createdAt)
	}
}

func TestGetConversationMessages_PaginatesBeyondFiftyMessages(t *testing.T) {
	newest := time.Date(2026, 7, 28, 1, 0, 0, 0, time.UTC)
	messages := make([]model.Message, 51)
	for i := range messages {
		messages[i] = model.Message{
			AMSeq:   9051 - i,
			RegDate: newest.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
		}
	}
	repo := &mockMessageRepo{canonicalMessages: messages}
	svc := &MessageService{repo: repo, profileRepo: &mockProfileRepo{}}

	resp, err := svc.GetConversationMessages(101, 202, "", 50)
	if err != nil {
		t.Fatalf("GetConversationMessages: %v", err)
	}
	if len(resp.Items) != 50 || !resp.HasMore || resp.NextCursor == nil {
		t.Fatalf("pagination = items:%d hasMore:%v cursor:%v", len(resp.Items), resp.HasMore, resp.NextCursor)
	}
	if repo.canonicalLimit != 51 {
		t.Fatalf("repository limit = %d, want 51", repo.canonicalLimit)
	}
}

func TestGetConversationMessages_RejectsMalformedCursorBeforeRepository(t *testing.T) {
	repo := &mockMessageRepo{}
	svc := &MessageService{repo: repo, profileRepo: &mockProfileRepo{}}

	_, err := svc.GetConversationMessages(101, 202, "not-an-opaque-cursor", 30)
	var validationErr *model.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	if repo.canonicalLimit != 0 {
		t.Fatalf("repository was called with limit %d", repo.canonicalLimit)
	}
}

func TestGetConversationMessages_NilBecomesEmptySlice(t *testing.T) {
	svc := &MessageService{repo: &mockMessageRepo{}, profileRepo: &mockProfileRepo{}}

	resp, err := svc.GetConversationMessages(101, 202, "", 30)
	if err != nil {
		t.Fatalf("GetConversationMessages: %v", err)
	}
	if resp.Items == nil || len(resp.Items) != 0 {
		t.Fatalf("items = %#v, want non-nil empty slice", resp.Items)
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
	if notifier.readThroughID != 100 {
		t.Fatalf("notification throughMessageId = %d, want 100", notifier.readThroughID)
	}
	if parsed, err := time.Parse(time.RFC3339, notifier.readAt); err != nil || parsed.Location() != time.UTC {
		t.Fatalf("notification readAt = %q, want UTC RFC3339", notifier.readAt)
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
	repo := &mockMessageRepo{markConvChanged: true}
	svc := &MessageService{
		repo:        repo,
		profileRepo: &mockProfileRepo{},
		notifier:    notifier,
	}

	if err := svc.MarkConversationRead(3, 7, 9001); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notifier.readCalls != 1 {
		t.Fatalf("expected one read notification, got %d", notifier.readCalls)
	}
	if notifier.readSenderSeq != 7 || notifier.readReaderSeq != 3 {
		t.Fatalf("unexpected notification payload: sender=%d reader=%d", notifier.readSenderSeq, notifier.readReaderSeq)
	}
	if notifier.readThroughID != 9001 {
		t.Fatalf("notification throughMessageId = %d, want 9001", notifier.readThroughID)
	}
	if parsed, err := time.Parse(time.RFC3339, notifier.readAt); err != nil || parsed.Location() != time.UTC {
		t.Fatalf("notification readAt = %q, want UTC RFC3339", notifier.readAt)
	}
	if repo.markConvThrough != 9001 {
		t.Fatalf("repository throughMessageId = %d, want 9001", repo.markConvThrough)
	}
}
