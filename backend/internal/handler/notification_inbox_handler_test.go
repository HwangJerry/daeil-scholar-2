package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

type notificationInboxServicerStub struct {
	userSeq, beforeSeq, size int
	listCalls, seenCalls     int
	response                 *model.NotificationListResponse
	err                      error
}

func (s *notificationInboxServicerStub) List(userSeq, beforeSeq, size int) (*model.NotificationListResponse, error) {
	s.listCalls++
	s.userSeq, s.beforeSeq, s.size = userSeq, beforeSeq, size
	return s.response, s.err
}

func (s *notificationInboxServicerStub) MarkAllSeen(userSeq int) error {
	s.seenCalls++
	s.userSeq = userSeq
	return s.err
}

func TestNotificationInboxHandlerListsWithCanonicalShape(t *testing.T) {
	cursor := "seq_500"
	stub := &notificationInboxServicerStub{response: &model.NotificationListResponse{
		Items: []model.NotificationItem{{
			ID: "notice-501", Type: "admin.notice", Title: "새 소식", Body: "장학금 안내",
			CreatedAt: "2026-09-28T01:00:00Z", IsUnread: true, DeepLink: "/feed/501", PostSeq: 501,
		}},
		NextCursor: &cursor, HasMore: true,
	}}
	recorder := httptest.NewRecorder()
	NewNotificationInboxHandler(stub).List(recorder, authenticatedPushRequest(http.MethodGet, "/api/notifications?cursor=seq_502&size=5", ""))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	want := `{"items":[{"id":"notice-501","type":"admin.notice","title":"새 소식","body":"장학금 안내","createdAt":"2026-09-28T01:00:00Z","isUnread":true,"deepLink":"/feed/501","postSeq":501}],"nextCursor":"seq_500","hasMore":true}` + "\n"
	if recorder.Body.String() != want {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if stub.userSeq != 42 || stub.beforeSeq != 502 || stub.size != 5 {
		t.Fatalf("service called with user %d before %d size %d", stub.userSeq, stub.beforeSeq, stub.size)
	}
}

func TestNotificationInboxHandlerTreatsMissingCursorAsFirstPage(t *testing.T) {
	stub := &notificationInboxServicerStub{response: &model.NotificationListResponse{Items: []model.NotificationItem{}}}
	recorder := httptest.NewRecorder()
	NewNotificationInboxHandler(stub).List(recorder, authenticatedPushRequest(http.MethodGet, "/api/notifications", ""))

	if recorder.Code != http.StatusOK || stub.beforeSeq != 0 || stub.size != 0 {
		t.Fatalf("status %d before %d size %d", recorder.Code, stub.beforeSeq, stub.size)
	}
	if recorder.Body.String() != `{"items":[],"nextCursor":null,"hasMore":false}`+"\n" {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestNotificationInboxHandlerRejectsInvalidCursor(t *testing.T) {
	for _, cursor := range []string{"abc", "seq_", "seq_-1", "seq_0", "0", "seq_1x"} {
		stub := &notificationInboxServicerStub{}
		recorder := httptest.NewRecorder()
		NewNotificationInboxHandler(stub).List(recorder, authenticatedPushRequest(http.MethodGet, "/api/notifications?cursor="+cursor, ""))

		if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"code":"INVALID_CURSOR"`) || stub.listCalls != 0 {
			t.Fatalf("cursor %q: status %d body %s", cursor, recorder.Code, recorder.Body.String())
		}
	}
}

func TestNotificationInboxHandlerRequiresAuthentication(t *testing.T) {
	stub := &notificationInboxServicerStub{}
	handler := NewNotificationInboxHandler(stub)
	for _, call := range []func(http.ResponseWriter, *http.Request){handler.List, handler.MarkSeen} {
		recorder := httptest.NewRecorder()
		call(recorder, httptest.NewRequest(http.MethodGet, "/api/notifications", nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", recorder.Code)
		}
	}
	if stub.listCalls != 0 || stub.seenCalls != 0 {
		t.Fatal("unauthenticated requests must not reach the service")
	}
}

func TestNotificationInboxHandlerListFailureIs500(t *testing.T) {
	stub := &notificationInboxServicerStub{err: errors.New("db down")}
	recorder := httptest.NewRecorder()
	NewNotificationInboxHandler(stub).List(recorder, authenticatedPushRequest(http.MethodGet, "/api/notifications", ""))

	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), `"code":"NOTIFICATIONS_FAILED"`) {
		t.Fatalf("status %d body %s", recorder.Code, recorder.Body.String())
	}
}

func TestNotificationInboxHandlerMarksSeenWithNoContent(t *testing.T) {
	stub := &notificationInboxServicerStub{}
	recorder := httptest.NewRecorder()
	NewNotificationInboxHandler(stub).MarkSeen(recorder, authenticatedPushRequest(http.MethodPost, "/api/notifications/seen", ""))

	if recorder.Code != http.StatusNoContent || recorder.Body.Len() != 0 || stub.seenCalls != 1 || stub.userSeq != 42 {
		t.Fatalf("status %d body %q calls %d user %d", recorder.Code, recorder.Body.String(), stub.seenCalls, stub.userSeq)
	}

	stub.err = errors.New("db down")
	recorder = httptest.NewRecorder()
	NewNotificationInboxHandler(stub).MarkSeen(recorder, authenticatedPushRequest(http.MethodPost, "/api/notifications/seen", ""))
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), `"code":"NOTIFICATIONS_SEEN_FAILED"`) {
		t.Fatalf("status %d body %s", recorder.Code, recorder.Body.String())
	}
}
