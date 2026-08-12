// password_reset_service_test.go — Unit tests for PasswordResetService ConfirmReset flow
package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
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
	insertedToken   string
	markUsedErr     error
	updatePwdErr    error
	updatePwdCalled bool
	updatedHash     string
	markUsedCalled  bool
	memberName      string
	memberNameErr   error
	tokenByLookup   map[string]*model.PasswordResetToken
	lookupTokens    []string
}

type mockAtomicPasswordResetRepo struct {
	mockPasswordResetRepo
	atomicCalls int
	tokenHash   string
	legacyToken string
	legacyHash  string
	replacement model.PasswordCredential
	resetEmail  string
}

func (m *mockAtomicPasswordResetRepo) FindPasswordResetAccountByVerifiedEmail(email string) (*model.User, error) {
	m.resetEmail = email
	return m.FindMemberByEmail(email)
}

func (m *mockAtomicPasswordResetRepo) ConfirmResetAtomically(tokenHash, legacyToken, legacyHash string, replacement model.PasswordCredential) error {
	m.atomicCalls++
	m.tokenHash = tokenHash
	m.legacyToken = legacyToken
	m.legacyHash = legacyHash
	m.replacement = replacement
	return nil
}

type concurrentAtomicPasswordResetRepo struct {
	mockPasswordResetRepo
	mu       sync.Mutex
	consumed bool
}

func (m *concurrentAtomicPasswordResetRepo) FindPasswordResetAccountByVerifiedEmail(email string) (*model.User, error) {
	return m.FindMemberByEmail(email)
}

func (m *concurrentAtomicPasswordResetRepo) ConfirmResetAtomically(string, string, string, model.PasswordCredential) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.consumed {
		return repository.ErrPasswordResetTokenInvalid
	}
	m.consumed = true
	return nil
}

func (m *mockPasswordResetRepo) FindMemberByEmail(email string) (*model.User, error) {
	return m.member, m.memberErr
}

func (m *mockPasswordResetRepo) InsertToken(usrSeq int, token string, expiresAt time.Time) error {
	m.insertedToken = token
	return m.insertTokenErr
}

func (m *mockPasswordResetRepo) FindValidToken(token string) (*model.PasswordResetToken, error) {
	m.lookupTokens = append(m.lookupTokens, token)
	if m.tokenByLookup != nil {
		return m.tokenByLookup[token], m.tokenErr
	}
	return m.token, m.tokenErr
}

func (m *mockPasswordResetRepo) MarkTokenUsed(token string) error {
	m.markUsedCalled = true
	return m.markUsedErr
}

func (m *mockPasswordResetRepo) UpdatePassword(usrSeq int, hashedPwd string) error {
	m.updatePwdCalled = true
	m.updatedHash = hashedPwd
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
		NewPassword: "newpass123!",
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
		NewPassword: "newpass123!",
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
		NewPassword: "newpass123!",
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
		NewPassword: "newpass123!",
	})
	if err == nil {
		t.Fatal("expected error from UpdatePassword failure")
	}
}

func TestConfirmResetWritesCanonicalAndLegacyPassword(t *testing.T) {
	repo := &mockPasswordResetRepo{
		token: &model.PasswordResetToken{USRSeq: 42, Token: "valid-token", ExpiresAt: time.Now().Add(time.Minute)},
	}
	credentials := &stubPasswordCredentialManager{}
	service := NewPasswordResetServiceWithPasswordCredentials(repo, credentials, make(chan model.EmailMessage, 1), zerolog.Nop(), "http://localhost")

	err := service.ConfirmReset(model.PasswordResetConfirm{Token: "valid-token", NewPassword: "Replacement123!"})
	if err != nil {
		t.Fatalf("ConfirmReset() error = %v", err)
	}
	if credentials.replaceCalls != 1 || credentials.replacedAccountSeq != 42 {
		t.Fatalf("canonical replacement = %+v", credentials)
	}
	if repo.updatedHash != MysqlNativePassword("Replacement123!") {
		t.Fatal("ConfirmReset() did not preserve the legacy fallback hash")
	}
}

func TestConfirmResetUsesAtomicRepositoryAndHashedToken(t *testing.T) {
	repo := &mockAtomicPasswordResetRepo{}
	service := NewAtomicPasswordResetService(repo, make(chan model.EmailMessage, 1), zerolog.Nop(), "http://localhost")

	err := service.ConfirmReset(model.PasswordResetConfirm{Token: "raw-reset-token", NewPassword: "Replacement123!"})
	if err != nil {
		t.Fatalf("ConfirmReset() error = %v", err)
	}
	if repo.atomicCalls != 1 || repo.tokenHash == "raw-reset-token" || len(repo.tokenHash) != 64 {
		t.Fatalf("atomic calls = %d, tokenHash = %q", repo.atomicCalls, repo.tokenHash)
	}
	if repo.legacyToken != "raw-reset-token" {
		t.Fatalf("legacy rollout token = %q", repo.legacyToken)
	}
	if repo.legacyHash != MysqlNativePassword("Replacement123!") || repo.replacement.Algorithm != model.PasswordAlgorithmArgon2id {
		t.Fatalf("legacy hash or canonical replacement missing: %+v", repo)
	}
}

func TestAtomicPasswordResetValidatesPreCutoverRawToken(t *testing.T) {
	rawToken := "pre-cutover-raw-token"
	repo := &mockAtomicPasswordResetRepo{mockPasswordResetRepo: mockPasswordResetRepo{
		tokenByLookup: map[string]*model.PasswordResetToken{
			rawToken: {USRSeq: 42, Token: rawToken, ExpiresAt: time.Now().Add(time.Minute)},
		},
		memberName: "Member",
	}}
	service := NewAtomicPasswordResetService(repo, make(chan model.EmailMessage, 1), zerolog.Nop(), "http://localhost")

	response, err := service.ValidateToken(rawToken)
	if err != nil {
		t.Fatal(err)
	}
	if !response.Valid || response.Name != "Member" {
		t.Fatalf("ValidateToken() = %+v", response)
	}
	if len(repo.lookupTokens) != 2 || repo.lookupTokens[0] != hashPasswordResetToken(rawToken) || repo.lookupTokens[1] != rawToken {
		t.Fatalf("token lookup order = %#v", repo.lookupTokens)
	}
}

func TestAtomicPasswordResetPersistsOnlyTokenHash(t *testing.T) {
	repo := &mockAtomicPasswordResetRepo{mockPasswordResetRepo: mockPasswordResetRepo{member: &model.User{USRSeq: 42, USRName: "Member"}}}
	service := NewAtomicPasswordResetService(repo, make(chan model.EmailMessage, 1), zerolog.Nop(), "http://localhost")

	if err := service.RequestReset(model.PasswordResetRequest{Email: " Member@Example.com "}); err != nil {
		t.Fatal(err)
	}
	if repo.resetEmail != "member@example.com" {
		t.Fatalf("canonical reset email = %q", repo.resetEmail)
	}
	if len(repo.insertedToken) != 64 {
		t.Fatalf("persisted token length = %d, want SHA-256 hex", len(repo.insertedToken))
	}
}

func TestConcurrentAtomicPasswordResetOnlyOneConfirmationSucceeds(t *testing.T) {
	repo := &concurrentAtomicPasswordResetRepo{}
	service := NewAtomicPasswordResetService(repo, make(chan model.EmailMessage, 1), zerolog.Nop(), "http://localhost")

	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			results <- service.ConfirmReset(model.PasswordResetConfirm{Token: "same-token", NewPassword: "Replacement123!"})
		}()
	}
	close(start)
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful confirmations = %d, want 1", successes)
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
