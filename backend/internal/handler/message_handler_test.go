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
	sendErr        error
	inboxResult    *model.MessageListResponse
	inboxErr       error
	outboxResult   *model.MessageListResponse
	outboxErr      error
	markAsReadErr  error
	deleteErr      error
	convResult     *model.ConversationListResponse
	convErr        error
	convMsgsResult *model.MessageListResponse
	convMsgsErr    error
	markConvErr    error
}

func (s *stubMsgService) SendMessage(_ int, _ string, _ model.SendMessageRequest) error {
	return s.sendErr
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
func (s *stubMsgService) MarkAsRead(_, _ int) error   { return s.markAsReadErr }
func (s *stubMsgService) DeleteMessage(_, _ int) error { return s.deleteErr }
func (s *stubMsgService) GetConversations(_ int) (*model.ConversationListResponse, error) {
	if s.convResult == nil {
		return &model.ConversationListResponse{Items: []model.ConversationSummary{}}, s.convErr
	}
	return s.convResult, s.convErr
}
func (s *stubMsgService) GetConversationMessages(_, _, _, _ int) (*model.MessageListResponse, error) {
	if s.convMsgsResult == nil {
		return &model.MessageListResponse{Items: []model.Message{}}, s.convMsgsErr
	}
	return s.convMsgsResult, s.convMsgsErr
}
func (s *stubMsgService) MarkConversationRead(_, _ int) error { return s.markConvErr }

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
	var resp map[string]string
	decodeJSON(t, rr, &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", resp["status"])
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

func TestGetConversations_ResponseShape(t *testing.T) {
	h := newTestHandler(&stubMsgService{
		convResult: &model.ConversationListResponse{
			Items: []model.ConversationSummary{
				{OtherSeq: 2, OtherName: "Alice", LastMessage: "hey", UnreadCount: 3},
			},
		},
	})
	rr := httptest.NewRecorder()
	h.GetConversations(rr, authRequest(http.MethodGet, "/api/messages/conversations", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp model.ConversationListResponse
	decodeJSON(t, rr, &resp)
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].OtherSeq != 2 || resp.Items[0].UnreadCount != 3 {
		t.Errorf("unexpected conversation item: %+v", resp.Items[0])
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
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(authRequest(http.MethodGet, "/api/messages/conversations/2", nil), "userSeq", "2")
	rr := httptest.NewRecorder()
	h.GetConversationMessages(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
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

func TestMarkConversationRead_ServiceError(t *testing.T) {
	h := newTestHandler(&stubMsgService{markConvErr: errors.New("db down")})
	req := withChiParam(authRequest(http.MethodPut, "/api/messages/conversations/2/read", nil), "userSeq", "2")
	rr := httptest.NewRecorder()
	h.MarkConversationRead(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestMarkConversationRead_Success(t *testing.T) {
	h := newTestHandler(&stubMsgService{})
	req := withChiParam(authRequest(http.MethodPut, "/api/messages/conversations/2/read", nil), "userSeq", "2")
	rr := httptest.NewRecorder()
	h.MarkConversationRead(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
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
