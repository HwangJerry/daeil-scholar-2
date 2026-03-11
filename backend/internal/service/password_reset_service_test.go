// password_reset_service_test.go — Unit tests for PasswordResetService ConfirmReset flow
package service

import (
	"errors"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

var errDB = errors.New("database error")

// mockPasswordResetRepo implements repository.PasswordResetQuerier for testing.
type mockPasswordResetRepo struct {
	member          *model.User
	memberErr       error
	token           *model.PasswordResetToken
	tokenErr        error
	insertTokenErr  error
	markUsedErr     error
	updatePwdErr    error
	updatePwdCalled bool
	markUsedCalled  bool
	memberName      string
	memberNameErr   error
}

func (m *mockPasswordResetRepo) FindMemberByEmail(email string) (*model.User, error) {
	return m.member, m.memberErr
}

func (m *mockPasswordResetRepo) InsertToken(usrSeq int, token string, expiresAt time.Time) error {
	return m.insertTokenErr
}

func (m *mockPasswordResetRepo) FindValidToken(token string) (*model.PasswordResetToken, error) {
	return m.token, m.tokenErr
}

func (m *mockPasswordResetRepo) MarkTokenUsed(token string) error {
	m.markUsedCalled = true
	return m.markUsedErr
}

func (m *mockPasswordResetRepo) UpdatePassword(usrSeq int, hashedPwd string) error {
	m.updatePwdCalled = true
	return m.updatePwdErr
}

func (m *mockPasswordResetRepo) GetMemberNameBySeq(usrSeq int) (string, error) {
	return m.memberName, m.memberNameErr
}

func newTestPasswordResetService(repo *mockPasswordResetRepo) *PasswordResetService {
	emailQueue := make(chan model.EmailMessage, 10)
	logger := zerolog.Nop()
	return &PasswordResetService{
		repo:        repo,
		emailQueue:  emailQueue,
		logger:      logger,
		siteBaseURL: "http://localhost",
	}
}

func TestConfirmReset_EmptyToken(t *testing.T) {
	repo := &mockPasswordResetRepo{}
	svc := newTestPasswordResetService(repo)

	err := svc.ConfirmReset(model.PasswordResetConfirm{
		Token:       "",
		NewPassword: "newpassword",
	})
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestConfirmReset_PasswordTooShort(t *testing.T) {
	repo := &mockPasswordResetRepo{}
	svc := newTestPasswordResetService(repo)

	err := svc.ConfirmReset(model.PasswordResetConfirm{
		Token:       "valid-token",
		NewPassword: "ab",
	})
	if err == nil {
		t.Fatal("expected error for password shorter than 4 characters")
	}
}

func TestConfirmReset_InvalidOrExpiredToken(t *testing.T) {
	repo := &mockPasswordResetRepo{token: nil, tokenErr: nil}
	svc := newTestPasswordResetService(repo)

	err := svc.ConfirmReset(model.PasswordResetConfirm{
		Token:       "expired-token",
		NewPassword: "newpassword",
	})
	if err == nil {
		t.Fatal("expected error for invalid/expired token")
	}
}

func TestConfirmReset_TokenLookupError(t *testing.T) {
	repo := &mockPasswordResetRepo{tokenErr: errDB}
	svc := newTestPasswordResetService(repo)

	err := svc.ConfirmReset(model.PasswordResetConfirm{
		Token:       "some-token",
		NewPassword: "newpassword",
	})
	if err == nil {
		t.Fatal("expected error from token lookup failure")
	}
}

func TestConfirmReset_Success(t *testing.T) {
	repo := &mockPasswordResetRepo{
		token: &model.PasswordResetToken{
			APRSeq:    1,
			USRSeq:    42,
			Token:     "valid-token",
			UsedYN:    "N",
			ExpiresAt: time.Now().Add(30 * time.Minute),
		},
	}
	svc := newTestPasswordResetService(repo)

	err := svc.ConfirmReset(model.PasswordResetConfirm{
		Token:       "valid-token",
		NewPassword: "newpassword123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.updatePwdCalled {
		t.Error("expected UpdatePassword to be called")
	}
	if !repo.markUsedCalled {
		t.Error("expected MarkTokenUsed to be called")
	}
}

func TestConfirmReset_UpdatePasswordError(t *testing.T) {
	repo := &mockPasswordResetRepo{
		token: &model.PasswordResetToken{
			APRSeq:    1,
			USRSeq:    42,
			Token:     "valid-token",
			UsedYN:    "N",
			ExpiresAt: time.Now().Add(30 * time.Minute),
		},
		updatePwdErr: errDB,
	}
	svc := newTestPasswordResetService(repo)

	err := svc.ConfirmReset(model.PasswordResetConfirm{
		Token:       "valid-token",
		NewPassword: "newpassword123",
	})
	if err == nil {
		t.Fatal("expected error from UpdatePassword failure")
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	repo := &mockPasswordResetRepo{}
	svc := newTestPasswordResetService(repo)

	resp, err := svc.ValidateToken("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Valid {
		t.Error("expected Valid=false for empty token")
	}
}

func TestValidateToken_ValidTokenWithName(t *testing.T) {
	repo := &mockPasswordResetRepo{
		token: &model.PasswordResetToken{
			APRSeq:    1,
			USRSeq:    42,
			Token:     "valid-token",
			UsedYN:    "N",
			ExpiresAt: time.Now().Add(30 * time.Minute),
		},
		memberName: "TestUser",
	}
	svc := newTestPasswordResetService(repo)

	resp, err := svc.ValidateToken("valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Valid {
		t.Error("expected Valid=true")
	}
	if resp.Name != "TestUser" {
		t.Errorf("expected Name='TestUser', got '%s'", resp.Name)
	}
}
