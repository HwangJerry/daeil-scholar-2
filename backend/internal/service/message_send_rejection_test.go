// message_send_rejection_test.go — Send refusal codes, check ordering and thread recipientAvailable.
package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

type mockBlockReader struct {
	blockedByMe bool
	err         error
	calls       int
}

func (m *mockBlockReader) Get(_, blockedSeq int) (*model.MemberBlockState, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return &model.MemberBlockState{UserSeq: blockedSeq, BlockedByMe: m.blockedByMe}, nil
}

func requireRejection(t *testing.T, err error, code string) {
	t.Helper()
	var rejection *model.MessageSendRejection
	if !errors.As(err, &rejection) {
		t.Fatalf("error = %v, want MessageSendRejection %s", err, code)
	}
	if rejection.Code != code {
		t.Fatalf("code = %s, want %s", rejection.Code, code)
	}
}

func newRejectionTestService(repo *mockMessageRepo, blocks *mockBlockReader) *MessageService {
	return NewMessageService(repo, &mockProfileRepo{}, blocks, nil)
}

func TestSendMessage_InvalidRequestsAreMessageInvalid(t *testing.T) {
	cases := map[string]model.SendMessageRequest{
		"empty content":     {UserSeq: 2, ClientMessageID: "c", Content: ""},
		"content over 1000": {UserSeq: 2, ClientMessageID: "c", Content: strings.Repeat("가", 1001)},
		"empty clientId":    {UserSeq: 2, ClientMessageID: "", Content: "hi"},
		"clientId over 64":  {UserSeq: 2, ClientMessageID: strings.Repeat("a", 65), Content: "hi"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &mockMessageRepo{}
			_, err := newRejectionTestService(repo, &mockBlockReader{}).SendMessage(1, "Sender", req)
			requireRejection(t, err, model.MessageSendInvalid)
			if repo.insertCalled {
				t.Fatal("invalid request reached AcceptMessage")
			}
		})
	}
}

func TestSendMessage_FilteredContentIsContentRejected(t *testing.T) {
	repo := &mockMessageRepo{}
	_, err := newRejectionTestService(repo, &mockBlockReader{}).SendMessage(1, "Sender",
		model.SendMessageRequest{UserSeq: 2, ClientMessageID: "c", Content: "씨발"})
	requireRejection(t, err, model.MessageSendContentRejected)
	if repo.insertCalled {
		t.Fatal("rejected content was accepted")
	}
}

func TestSendMessage_UnavailableRecipientsAreRecipientUnavailable(t *testing.T) {
	cases := map[string]struct {
		recipient int
		approved  bool
	}{
		"no recipient":                  {recipient: 0, approved: true},
		"self":                          {recipient: 1, approved: true},
		"withdrawn or unapproved (AAA)": {recipient: 2, approved: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &mockMessageRepo{approvedSet: true, approved: tc.approved}
			blocks := &mockBlockReader{blockedByMe: true}
			_, err := newRejectionTestService(repo, blocks).SendMessage(1, "Sender",
				model.SendMessageRequest{UserSeq: tc.recipient, ClientMessageID: "c", Content: "hi"})
			requireRejection(t, err, model.MessageSendRecipientGone)
			if repo.insertCalled {
				t.Fatal("unavailable recipient reached AcceptMessage")
			}
		})
	}
}

func TestSendMessage_BlockedByMeIsRejectedBeforeAccept(t *testing.T) {
	repo := &mockMessageRepo{}
	blocks := &mockBlockReader{blockedByMe: true}
	_, err := newRejectionTestService(repo, blocks).SendMessage(1, "Sender",
		model.SendMessageRequest{UserSeq: 2, ClientMessageID: "c", Content: "hi"})
	requireRejection(t, err, model.MessageSendBlockedByMe)
	if repo.insertCalled {
		t.Fatal("blocked-by-me send reached AcceptMessage")
	}
}

func TestSendMessage_BlockLookupErrorIsInfrastructureError(t *testing.T) {
	_, err := newRejectionTestService(&mockMessageRepo{}, &mockBlockReader{err: errors.New("db down")}).SendMessage(1, "Sender",
		model.SendMessageRequest{UserSeq: 2, ClientMessageID: "c", Content: "hi"})
	var rejection *model.MessageSendRejection
	if err == nil || errors.As(err, &rejection) {
		t.Fatalf("error = %v, want plain infrastructure error", err)
	}
}

func TestSendMessage_ReplayReturnsOriginalBeforeBlockAndAvailabilityChecks(t *testing.T) {
	original := &model.SendMessageResponse{MessageID: 9001, ClientMessageID: "replay", Status: "accepted", CreatedAt: "2026-07-28T01:00:00Z"}
	repo := &mockMessageRepo{findResult: original, approvedSet: true, approved: false}
	blocks := &mockBlockReader{blockedByMe: true}
	accepted, err := newRejectionTestService(repo, blocks).SendMessage(1, "Sender",
		model.SendMessageRequest{UserSeq: 2, ClientMessageID: "replay", Content: "씨발"})
	if err != nil {
		t.Fatalf("replay error = %v, want original acceptance", err)
	}
	if accepted != original {
		t.Fatalf("accepted = %+v, want original", accepted)
	}
	if blocks.calls != 0 || repo.insertCalled {
		t.Fatalf("replay ran checks: blockCalls=%d insert=%v", blocks.calls, repo.insertCalled)
	}
}

func TestSendMessage_RecipientBlockedMeStaysShadowAccept(t *testing.T) {
	repo := &mockMessageRepo{acceptResult: &model.SendMessageResponse{
		MessageID: 9002, ClientMessageID: "c", Status: "accepted", WasCreated: true, VisibleToRecipient: "N",
	}}
	accepted, err := newRejectionTestService(repo, &mockBlockReader{}).SendMessage(1, "Sender",
		model.SendMessageRequest{UserSeq: 2, ClientMessageID: "c", Content: "hi"})
	if err != nil || accepted.Status != "accepted" {
		t.Fatalf("accepted = %+v err = %v, want shadow accept", accepted, err)
	}
}

func TestGetConversationMessages_RecipientAvailable(t *testing.T) {
	cases := map[string]struct {
		other    int
		approved bool
		want     bool
	}{
		"approved peer":        {other: 2, approved: true, want: true},
		"withdrawn peer (AAA)": {other: 2, approved: false, want: false},
		"unapproved peer":      {other: 3, approved: false, want: false},
		"self thread":          {other: 1, approved: true, want: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &mockMessageRepo{approvedSet: true, approved: tc.approved}
			result, err := newRejectionTestService(repo, &mockBlockReader{}).GetConversationMessages(1, tc.other, "", 30)
			if err != nil {
				t.Fatalf("GetConversationMessages: %v", err)
			}
			if result.RecipientAvailable != tc.want {
				t.Fatalf("recipientAvailable = %v, want %v", result.RecipientAvailable, tc.want)
			}
		})
	}
}

func TestGetConversationMessages_AvailabilityLookupErrorFails(t *testing.T) {
	repo := &mockMessageRepo{approvedErr: errors.New("db down")}
	if _, err := newRejectionTestService(repo, &mockBlockReader{}).GetConversationMessages(1, 2, "", 30); err == nil {
		t.Fatal("expected availability lookup error")
	}
}
