package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

type badgeCounterStub struct {
	messages, notifications     int
	messageErr, notificationErr error
}

func (s badgeCounterStub) GetUnreadCount(int) (int, error) { return s.messages, s.messageErr }
func (s badgeCounterStub) CountUnread(int) (int, error)    { return s.notifications, s.notificationErr }

func TestBadgeHandlerReportsMessagesAndNotifications(t *testing.T) {
	stub := badgeCounterStub{messages: 3, notifications: 2}
	recorder := httptest.NewRecorder()
	NewBadgeHandler(stub, stub, zerolog.Nop()).GetBadges(recorder, authenticatedPushRequest(http.MethodGet, "/api/badges", ""))

	if recorder.Code != http.StatusOK || recorder.Body.String() != `{"unreadMessages":3,"unreadNotifications":2}`+"\n" {
		t.Fatalf("status %d body %s", recorder.Code, recorder.Body.String())
	}
}

// A failing notification count must not take the message badge down with it.
func TestBadgeHandlerDegradesFailingCountToZero(t *testing.T) {
	stub := badgeCounterStub{messages: 3, notifications: 2, notificationErr: errors.New("db down")}
	recorder := httptest.NewRecorder()
	NewBadgeHandler(stub, stub, zerolog.Nop()).GetBadges(recorder, authenticatedPushRequest(http.MethodGet, "/api/badges", ""))

	if recorder.Code != http.StatusOK || recorder.Body.String() != `{"unreadMessages":3,"unreadNotifications":0}`+"\n" {
		t.Fatalf("status %d body %s", recorder.Code, recorder.Body.String())
	}
}
