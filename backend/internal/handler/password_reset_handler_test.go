// password_reset_handler_test.go — Unit tests for PasswordResetHandler HTTP endpoints
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

// stubPasswordResetRepo is a minimal PasswordResetQuerier stub for handler tests.
type stubPasswordResetRepo struct{}

func (s *stubPasswordResetRepo) FindMemberByEmail(email string) (*model.User, error) {
	return nil, nil
}
func (s *stubPasswordResetRepo) InsertToken(usrSeq int, token string, expiresAt time.Time) error {
	return nil
}
func (s *stubPasswordResetRepo) FindValidToken(token string) (*model.PasswordResetToken, error) {
	return nil, nil
}
func (s *stubPasswordResetRepo) MarkTokenUsed(token string) error          { return nil }
func (s *stubPasswordResetRepo) UpdatePassword(usrSeq int, hashed string) error { return nil }
func (s *stubPasswordResetRepo) GetMemberNameBySeq(usrSeq int) (string, error) {
	return "", nil
}

func newTestPasswordResetHandler() *PasswordResetHandler {
	emailCh := make(chan model.EmailMessage, 10)
	logger := zerolog.Nop()
	svc := service.NewPasswordResetService(&stubPasswordResetRepo{}, emailCh, logger, "http://localhost")
	return NewPasswordResetHandler(svc, logger)
}

func TestRequestReset_EmptyEmail(t *testing.T) {
	h := newTestPasswordResetHandler()

	body := strings.NewReader(`{"email":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/password/reset-request", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.RequestReset(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var apiErr model.APIError
	if err := json.NewDecoder(rr.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if apiErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected error code VALIDATION_ERROR, got %s", apiErr.Code)
	}
}

func TestRequestReset_MalformedJSON(t *testing.T) {
	h := newTestPasswordResetHandler()

	body := strings.NewReader(`{invalid json`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/password/reset-request", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.RequestReset(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var apiErr model.APIError
	if err := json.NewDecoder(rr.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if apiErr.Code != "INVALID_BODY" {
		t.Errorf("expected error code INVALID_BODY, got %s", apiErr.Code)
	}
}

func TestRequestReset_ValidEmail(t *testing.T) {
	h := newTestPasswordResetHandler()

	body := strings.NewReader(`{"email":"test@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/password/reset-request", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.RequestReset(rr, req)

	// Service returns nil for unknown emails (security: no email existence leak)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestConfirmReset_MalformedJSON(t *testing.T) {
	h := newTestPasswordResetHandler()

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/password/reset-confirm", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ConfirmReset(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var apiErr model.APIError
	if err := json.NewDecoder(rr.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if apiErr.Code != "INVALID_BODY" {
		t.Errorf("expected error code INVALID_BODY, got %s", apiErr.Code)
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	h := newTestPasswordResetHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/password/validate-token", nil)
	rr := httptest.NewRecorder()
	h.ValidateToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp model.ValidateTokenResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Valid {
		t.Error("expected valid to be false for empty token")
	}
}

func TestValidateToken_UnknownToken(t *testing.T) {
	h := newTestPasswordResetHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/password/validate-token?token=nonexistent", nil)
	rr := httptest.NewRecorder()
	h.ValidateToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp model.ValidateTokenResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Valid {
		t.Error("expected valid to be false for unknown token")
	}
}
