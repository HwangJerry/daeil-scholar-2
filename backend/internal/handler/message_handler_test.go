// message_handler_test.go — Unit tests for MessageHandler HTTP endpoints.
// The handler is tested in isolation by stubbing MessageServicer directly,
// without wiring up the real service or a repo underneath.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-chi/chi/v5"
)

// ── Stub service ──────────────────────────────────────────────────────────────

type stubMsgService struct {
	sendResult      *model.SendMessageResponse
	sendErr         error
	inboxResult     *model.MessageListResponse
	inboxErr        error
	outboxResult    *model.MessageListResponse
	outboxErr       error
	markAsReadErr   error
	deleteErr       error
	convResult      *model.ConversationListResponse
	convErr         error
	convMsgsResult  *model.ConversationMessageListResponse
	convMsgsErr     error
	markConvErr     error
	markConvThrough int64
}

func (s *stubMsgService) SendMessage(_ int, _ string, _ model.SendMessageRequest) (*model.SendMessageResponse, error) {
	if s.sendResult == nil {
		s.sendResult = &model.SendMessageResponse{
			MessageID:       9001,
			ClientMessageID: "018f1f1a-7c65-7b65-b845-123456789abc",
			Status:          "accepted",
			CreatedAt:       "2026-07-28T01:00:00Z",
		}
	}
	return s.sendResult, s.sendErr
}
func (s *stubMsgService) GetInbox(_, _, _ int) (*model.MessageListResponse, error) {
	if s.inboxResult == nil {
		return &model.MessageListResponse{Items: []model.Message{}}, s.inboxErr
	}
	return s.inboxResult, s.inboxErr
}
func (s *stubMsgService) GetOutbox(_, _, _ int) (*model.MessageListResponse, error) {
	if s.outboxResult == nil {
		return &model.MessageListResponse{Items: []model.Message{}}, s.outboxErr
	}
	return s.outboxResult, s.outboxErr
}
func (s *stubMsgService) MarkAsRead(_, _ int) error    { return s.markAsReadErr }
func (s *stubMsgService) DeleteMessage(_, _ int) error { return s.deleteErr }
func (s *stubMsgService) GetConversations(_ int, _ string, size int) (*model.ConversationListResponse, error) {
	if s.convResult == nil {
		return &model.ConversationListResponse{Items: []model.ConversationSummary{}}, s.convErr
	}
	if size > 0 && len(s.convResult.Items) > size {
		nextCursor := "opaque-next"
		return &model.ConversationListResponse{
			Items:      s.convResult.Items[:size],
			NextCursor: &nextCursor,
			HasMore:    true,
		}, s.convErr
	}
	return s.convResult, s.convErr
}
func (s *stubMsgService) GetConversationMessages(_, _ int, _ string, _ int) (*model.ConversationMessageListResponse, error) {
	if s.convMsgsResult == nil {
		return &model.ConversationMessageListResponse{Items: []model.ConversationMessage{}}, s.convMsgsErr
	}
	return s.convMsgsResult, s.convMsgsErr
}
func (s *stubMsgService) MarkConversationRead(_, _ int, throughMessageID int64) error {
	s.markConvThrough = throughMessageID
	return s.markConvErr
}

var _ MessageServicer = (*stubMsgService)(nil)

// ── Helpers ───────────────────────────────────────────────────────────────────

func newTestHandler(svc MessageServicer) *MessageHandler {
	return NewMessageHandler(svc)
}

func authRequest(method, target string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, target, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	ctx := middleware.SetAuthUser(req.Context(), &model.AuthUser{
		USRSeq:    1,
		USRID:     "tester",
		USRName:   "Test User",
		USRStatus: "BBB",
	})
	return req.WithContext(ctx)
}

func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder, dst interface{}) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(dst); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
}

// ── Send ──────────────────────────────────────────────────────────────────────

func TestMessageSend_Unauthorized(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	rr := httptest.NewRecorder()
	h.Send(rr, httptest.NewRequest(http.MethodPost, "/api/messages", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	var apiErr model.APIError
	decodeJSON(t, rr, &apiErr)
	if apiErr.Code != "UNAUTHORIZED" {
		t.Errorf("expected UNAUTHORIZED, got %s", apiErr.Code)
	}
}

func TestMessageSend_InvalidJSON(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	rr := httptest.NewRecorder()
	h.Send(rr, authRequest(http.MethodPost, "/api/messages", []byte("{bad json")))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var apiErr model.APIError
	decodeJSON(t, rr, &apiErr)
	if apiErr.Code != "INVALID_BODY" {
		t.Errorf("expected INVALID_BODY, got %s", apiErr.Code)
	}
}

func TestMessageSend_ValidationError(t *testing.T) {
	// Service returns ValidationError → handler must respond 400.
	h := newTestHandler(&stubMsgService{
		sendErr: &model.ValidationError{Msg: "메시지 내용을 입력해주세요"},
	})
	body, _ := json.Marshal(model.SendMessageRequest{RecvrSeq: 2, Content: ""})
	rr := httptest.NewRecorder()
	h.Send(rr, authRequest(http.MethodPost, "/api/messages", body))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var apiErr model.APIError
	decodeJSON(t, rr, &apiErr)
	if apiErr.Code != "SEND_FAILED" {
		t.Errorf("expected SEND_FAILED, got %s", apiErr.Code)
	}
}

func TestMessageSend_InfraError(t *testing.T) {
	// Non-ValidationError from service → handler must respond 500, not 400.
	h := newTestHandler(&stubMsgService{sendErr: errors.New("db down")})
	body, _ := json.Marshal(model.SendMessageRequest{RecvrSeq: 2, Content: "Hello"})
	rr := httptest.NewRecorder()
	h.Send(rr, authRequest(http.MethodPost, "/api/messages", body))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestMessageSend_Success(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	body, _ := json.Marshal(model.SendMessageRequest{RecvrSeq: 2, Content: "Hello!"})
	rr := httptest.NewRecorder()
	h.Send(rr, authRequest(http.MethodPost, "/api/messages", body))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp model.SendMessageResponse
	decodeJSON(t, rr, &resp)
	if resp.Status != "accepted" {
		t.Errorf("expected status=accepted, got %s", resp.Status)
	}
}

func TestMessageSend_ReturnsCanonicalAcceptedResponse(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	body := []byte(`{"userSeq":202,"clientMessageId":"018f1f1a-7c65-7b65-b845-123456789abc","content":"안녕하세요."}`)
	rr := httptest.NewRecorder()
	h.Send(rr, authRequest(http.MethodPost, "/api/messages", body))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var response map[string]interface{}
	decodeJSON(t, rr, &response)
	if response["messageId"] != float64(9001) {
		t.Fatalf("messageId = %#v, want 9001", response["messageId"])
	}
	if response["clientMessageId"] != "018f1f1a-7c65-7b65-b845-123456789abc" {
		t.Fatalf("clientMessageId = %#v", response["clientMessageId"])
	}
	if response["status"] != "accepted" {
		t.Fatalf("status = %#v, want accepted", response["status"])
	}
	if response["createdAt"] != "2026-07-28T01:00:00Z" {
		t.Fatalf("createdAt = %#v", response["createdAt"])
	}
	if len(response) != 4 {
		t.Fatalf("response fields = %#v, want closed canonical four-field response", response)
	}
}

// ── GetInbox ──────────────────────────────────────────────────────────────────

func TestMessageGetInbox_Unauthorized(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	rr := httptest.NewRecorder()
	h.GetInbox(rr, httptest.NewRequest(http.MethodGet, "/api/messages/inbox", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestMessageGetInbox_ServiceError(t *testing.T) {
	h := newTestHandler(&stubMsgService{inboxErr: errors.New("db down")})
	rr := httptest.NewRecorder()
	h.GetInbox(rr, authRequest(http.MethodGet, "/api/messages/inbox", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
	var apiErr model.APIError
	decodeJSON(t, rr, &apiErr)
	if apiErr.Code != "INBOX_FAILED" {
		t.Errorf("expected INBOX_FAILED, got %s", apiErr.Code)
	}
}

func TestMessageGetInbox_ResponseShape(t *testing.T) {
	h := newTestHandler(&stubMsgService{
		inboxResult: &model.MessageListResponse{
			Items:      []model.Message{{AMSeq: 1, Content: "hi", SenderSeq: 2, RecvrSeq: 1}},
			TotalCount: 1,
			Page:       1,
			Size:       20,
			TotalPages: 1,
		},
	})
	rr := httptest.NewRecorder()
	h.GetInbox(rr, authRequest(http.MethodGet, "/api/messages/inbox", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp model.MessageListResponse
	decodeJSON(t, rr, &resp)
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].AMSeq != 1 {
		t.Errorf("expected amSeq=1, got %d", resp.Items[0].AMSeq)
	}
	if resp.TotalCount != 1 {
		t.Errorf("expected totalCount=1, got %d", resp.TotalCount)
	}
	if resp.Page != 1 || resp.Size != 20 || resp.TotalPages != 1 {
		t.Errorf("unexpected pagination: page=%d size=%d totalPages=%d", resp.Page, resp.Size, resp.TotalPages)
	}
}

// ── GetOutbox ─────────────────────────────────────────────────────────────────

func TestMessageGetOutbox_Unauthorized(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	rr := httptest.NewRecorder()
	h.GetOutbox(rr, httptest.NewRequest(http.MethodGet, "/api/messages/outbox", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestMessageGetOutbox_ServiceError(t *testing.T) {
	h := newTestHandler(&stubMsgService{outboxErr: errors.New("db down")})
	rr := httptest.NewRecorder()
	h.GetOutbox(rr, authRequest(http.MethodGet, "/api/messages/outbox", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
	var apiErr model.APIError
	decodeJSON(t, rr, &apiErr)
	if apiErr.Code != "OUTBOX_FAILED" {
		t.Errorf("expected OUTBOX_FAILED, got %s", apiErr.Code)
	}
}

func TestMessageGetOutbox_ResponseShape(t *testing.T) {
	h := newTestHandler(&stubMsgService{
		outboxResult: &model.MessageListResponse{
			Items:      []model.Message{{AMSeq: 5, Content: "sent", SenderSeq: 1, RecvrSeq: 2}},
			TotalCount: 1,
			Page:       1,
			Size:       20,
			TotalPages: 1,
		},
	})
	rr := httptest.NewRecorder()
	h.GetOutbox(rr, authRequest(http.MethodGet, "/api/messages/outbox", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp model.MessageListResponse
	decodeJSON(t, rr, &resp)
	if len(resp.Items) != 1 || resp.Items[0].AMSeq != 5 {
		t.Errorf("unexpected outbox response: %+v", resp)
	}
}

// ── MarkAsRead ────────────────────────────────────────────────────────────────

func TestMessageMarkAsRead_Unauthorized(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(httptest.NewRequest(http.MethodPut, "/api/messages/1/read", nil), "seq", "1")
	rr := httptest.NewRecorder()
	h.MarkAsRead(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestMessageMarkAsRead_InvalidSeq(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(authRequest(http.MethodPut, "/api/messages/abc/read", nil), "seq", "abc")
	rr := httptest.NewRecorder()
	h.MarkAsRead(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var apiErr model.APIError
	decodeJSON(t, rr, &apiErr)
	if apiErr.Code != "INVALID_SEQ" {
		t.Errorf("expected INVALID_SEQ, got %s", apiErr.Code)
	}
}

func TestMessageMarkAsRead_ServiceError(t *testing.T) {
	h := newTestHandler(&stubMsgService{markAsReadErr: errors.New("not found")})
	req := withChiParam(authRequest(http.MethodPut, "/api/messages/1/read", nil), "seq", "1")
	rr := httptest.NewRecorder()
	h.MarkAsRead(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestMessageMarkAsRead_Success(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(authRequest(http.MethodPut, "/api/messages/5/read", nil), "seq", "5")
	rr := httptest.NewRecorder()
	h.MarkAsRead(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestMessageDelete_InvalidSeq(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(authRequest(http.MethodDelete, "/api/messages/xyz", nil), "seq", "xyz")
	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestMessageDelete_ServiceError(t *testing.T) {
	h := newTestHandler(&stubMsgService{deleteErr: errors.New("not found")})
	req := withChiParam(authRequest(http.MethodDelete, "/api/messages/3", nil), "seq", "3")
	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestMessageDelete_Success(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(authRequest(http.MethodDelete, "/api/messages/3", nil), "seq", "3")
	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

// ── GetConversations ──────────────────────────────────────────────────────────

func TestGetConversations_Unauthorized(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	rr := httptest.NewRecorder()
	h.GetConversations(rr, httptest.NewRequest(http.MethodGet, "/api/messages/conversations", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetConversations_ServiceError(t *testing.T) {
	h := newTestHandler(&stubMsgService{convErr: errors.New("db down")})
	rr := httptest.NewRecorder()
	h.GetConversations(rr, authRequest(http.MethodGet, "/api/messages/conversations", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
	var apiErr model.APIError
	decodeJSON(t, rr, &apiErr)
	if apiErr.Code != "CONVERSATIONS_FAILED" {
		t.Errorf("expected CONVERSATIONS_FAILED, got %s", apiErr.Code)
	}
}

func TestGetConversations_InvalidCursorReturnsBadRequest(t *testing.T) {
	h := newTestHandler(&stubMsgService{convErr: &model.ValidationError{Msg: "cursor가 올바르지 않습니다"}})
	rr := httptest.NewRecorder()
	h.GetConversations(rr, authRequest(http.MethodGet, "/api/messages/conversations?cursor=bad", nil))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var apiErr model.APIError
	decodeJSON(t, rr, &apiErr)
	if apiErr.Code != "INVALID_REQUEST" {
		t.Fatalf("code = %q, want INVALID_REQUEST", apiErr.Code)
	}
}

func TestGetConversations_ResponseShape(t *testing.T) {
	h := newTestHandler(&stubMsgService{
		convResult: &model.ConversationListResponse{
			Items: []model.ConversationSummary{
				{UserSeq: 2, Name: "Alice", LastMessage: "hey", LastMessageAt: "2026-07-28T01:00:00Z", UnreadCount: 3},
				{UserSeq: 3, Name: "Bob", LastMessage: "older", LastMessageAt: "2026-07-27T01:00:00Z"},
			},
		},
	})
	rr := httptest.NewRecorder()
	h.GetConversations(rr, authRequest(http.MethodGet, "/api/messages/conversations?size=1", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	decodeJSON(t, rr, &resp)
	items, ok := resp["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v, want one canonical item", resp["items"])
	}
	item := items[0].(map[string]interface{})
	if item["userSeq"] != float64(2) || item["name"] != "Alice" || item["lastMessageAt"] != "2026-07-28T01:00:00Z" {
		t.Fatalf("canonical item = %#v", item)
	}
	if item["blockedByMe"] != false || item["unreadCount"] != float64(3) {
		t.Fatalf("canonical state = %#v", item)
	}
	if resp["hasMore"] != true || resp["nextCursor"] == nil {
		t.Fatalf("pagination = %#v", resp)
	}
}

// ── GetConversationMessages ───────────────────────────────────────────────────

func TestGetConversationMessages_InvalidUserSeq(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(authRequest(http.MethodGet, "/api/messages/conversations/bad", nil), "userSeq", "bad")
	rr := httptest.NewRecorder()
	h.GetConversationMessages(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetConversationMessages_RejectsNonPositiveUserSeq(t *testing.T) {
	h := newTestHandler(&stubMsgService{convMsgsResult: &model.ConversationMessageListResponse{Items: []model.ConversationMessage{}}})
	req := withChiParam(authRequest(http.MethodGet, "/api/messages/conversations/0", nil), "userSeq", "0")
	rr := httptest.NewRecorder()
	h.GetConversationMessages(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestGetConversationMessages_ServiceError(t *testing.T) {
	h := newTestHandler(&stubMsgService{convMsgsErr: errors.New("db down")})
	req := withChiParam(authRequest(http.MethodGet, "/api/messages/conversations/2", nil), "userSeq", "2")
	rr := httptest.NewRecorder()
	h.GetConversationMessages(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
	var apiErr model.APIError
	decodeJSON(t, rr, &apiErr)
	if apiErr.Code != "CONVERSATION_MESSAGES_FAILED" {
		t.Errorf("expected CONVERSATION_MESSAGES_FAILED, got %s", apiErr.Code)
	}
}

func TestGetConversationMessages_Success(t *testing.T) {
	h := newTestHandler(&stubMsgService{convMsgsResult: &model.ConversationMessageListResponse{Items: []model.ConversationMessage{
		{
			MessageID: 9001, Sender: model.MessageParticipant{UserSeq: 1, Name: "Test User"},
			RecipientUserSeq: 2, Content: "hello", Read: true,
			CreatedAt: "2026-07-28T01:00:00Z",
		},
	}}})
	req := withChiParam(authRequest(http.MethodGet, "/api/messages/conversations/2", nil), "userSeq", "2")
	rr := httptest.NewRecorder()
	h.GetConversationMessages(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	decodeJSON(t, rr, &resp)
	items, ok := resp["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v", resp["items"])
	}
	item := items[0].(map[string]interface{})
	sender, ok := item["sender"].(map[string]interface{})
	if !ok || sender["userSeq"] != float64(1) || sender["name"] != "Test User" {
		t.Fatalf("sender = %#v", item["sender"])
	}
	if item["messageId"] != float64(9001) || item["recipientUserSeq"] != float64(2) || item["read"] != true {
		t.Fatalf("canonical message = %#v", item)
	}
	if _, legacy := resp["totalCount"]; legacy || resp["hasMore"] != false {
		t.Fatalf("canonical envelope = %#v", resp)
	}
}

// ── MarkConversationRead ──────────────────────────────────────────────────────

func TestMarkConversationRead_Unauthorized(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(httptest.NewRequest(http.MethodPut, "/api/messages/conversations/2/read", nil), "userSeq", "2")
	rr := httptest.NewRecorder()
	h.MarkConversationRead(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestMarkConversationRead_InvalidSeq(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(authRequest(http.MethodPut, "/api/messages/conversations/abc/read", nil), "userSeq", "abc")
	rr := httptest.NewRecorder()
	h.MarkConversationRead(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var apiErr model.APIError
	decodeJSON(t, rr, &apiErr)
	if apiErr.Code != "INVALID_SEQ" {
		t.Errorf("expected INVALID_SEQ, got %s", apiErr.Code)
	}
}

func TestMarkConversationRead_RejectsNonPositiveUserSeq(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(authRequest(http.MethodPut, "/api/messages/conversations/0/read", []byte(`{"throughMessageId":9001}`)), "userSeq", "0")
	rr := httptest.NewRecorder()
	h.MarkConversationRead(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestMarkConversationRead_ServiceError(t *testing.T) {
	h := newTestHandler(&stubMsgService{markConvErr: errors.New("db down")})
	req := withChiParam(authRequest(http.MethodPut, "/api/messages/conversations/2/read", []byte(`{"throughMessageId":9001}`)), "userSeq", "2")
	rr := httptest.NewRecorder()
	h.MarkConversationRead(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestMarkConversationRead_Success(t *testing.T) {
	stub := &stubMsgService{}
	h := newTestHandler(stub)
	req := withChiParam(authRequest(http.MethodPut, "/api/messages/conversations/2/read", []byte(`{"throughMessageId":9001}`)), "userSeq", "2")
	rr := httptest.NewRecorder()
	h.MarkConversationRead(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("204 response body = %q, want empty", rr.Body.String())
	}
	if stub.markConvThrough != 9001 {
		t.Fatalf("throughMessageId = %d, want 9001", stub.markConvThrough)
	}
}

func TestMarkConversationRead_RequiresThroughMessageID(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(authRequest(http.MethodPut, "/api/messages/conversations/2/read", []byte(`{}`)), "userSeq", "2")
	rr := httptest.NewRecorder()
	h.MarkConversationRead(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var apiErr model.APIError
	decodeJSON(t, rr, &apiErr)
	if apiErr.Code != "INVALID_REQUEST" {
		t.Fatalf("code = %q, want INVALID_REQUEST", apiErr.Code)
	}
}

// ── parsePagination ───────────────────────────────────────────────────────────

func TestParsePagination_Defaults(t *testing.T) {
	page, size := parsePagination(httptest.NewRequest(http.MethodGet, "/", nil))
	if page != 1 || size != 20 {
		t.Errorf("expected (1, 20), got (%d, %d)", page, size)
	}
}

func TestParsePagination_CustomValues(t *testing.T) {
	page, size := parsePagination(httptest.NewRequest(http.MethodGet, "/?page=3&size=10", nil))
	if page != 3 || size != 10 {
		t.Errorf("expected (3, 10), got (%d, %d)", page, size)
	}
}

func TestParsePagination_InvalidFallsToDefault(t *testing.T) {
	page, size := parsePagination(httptest.NewRequest(http.MethodGet, "/?page=abc&size=-1", nil))
	if page != 1 || size != 20 {
		t.Errorf("expected (1, 20), got (%d, %d)", page, size)
	}
}
