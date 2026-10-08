// message_receiving_test.go — Receiving opt-out blocks sends while preserving history and idempotency.
package service

import (
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"testing"
)

func TestReceivingDisabledRejectsBeforeInsertAndCanBeReenabled(t *testing.T) {
	repo := &mockMessageRepo{receiveDisabled: true}
	svc := newRejectionTestService(repo, &mockBlockReader{})
	req := model.SendMessageRequest{UserSeq: 2, ClientMessageID: "new", Content: "안녕하세요"}
	_, err := svc.SendMessage(1, "Sender", req)
	requireRejection(t, err, model.MessageSendReceivingDisabled)
	if repo.insertCalled {
		t.Fatal("disabled recipient got a new message")
	}
	repo.receiveDisabled = false
	if _, err := svc.SendMessage(1, "Sender", req); err != nil || !repo.insertCalled {
		t.Fatalf("reenabled send: %v", err)
	}
}

func TestReceivingDisabledKeepsHistoryButDisablesComposer(t *testing.T) {
	repo := &mockMessageRepo{receiveDisabled: true, canonicalMessages: []model.Message{{AMSeq: 1, SenderSeq: 2, RecvrSeq: 1, Content: "이전 쪽지", RegDate: "2026-07-28T01:00:00Z"}}}
	response, err := newRejectionTestService(repo, &mockBlockReader{}).GetConversationMessages(1, 2, "", 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 1 || !response.RecipientAvailable || response.RecipientMessageAllowed {
		t.Fatalf("response: %+v", response)
	}
}

func TestReceivingDisabledPreservesPreviouslyAcceptedReplay(t *testing.T) {
	original := &model.SendMessageResponse{MessageID: 7, ClientMessageID: "old", Status: "accepted"}
	repo := &mockMessageRepo{receiveDisabled: true, receiveErr: errors.New("must not read preference"), findResult: original}
	got, err := newRejectionTestService(repo, &mockBlockReader{}).SendMessage(1, "Sender", model.SendMessageRequest{UserSeq: 2, ClientMessageID: "old", Content: "안녕하세요"})
	if err != nil || got != original || repo.insertCalled {
		t.Fatalf("replay: %+v, %v", got, err)
	}
}

func TestReceivingPreferenceLookupFailureFailsClosed(t *testing.T) {
	repo := &mockMessageRepo{receiveErr: errors.New("db down")}
	_, err := newRejectionTestService(repo, &mockBlockReader{}).SendMessage(1, "Sender", model.SendMessageRequest{UserSeq: 2, ClientMessageID: "new", Content: "안녕하세요"})
	if err == nil || repo.insertCalled {
		t.Fatal("send continued without a receiving preference")
	}
}
