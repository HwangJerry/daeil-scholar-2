package service

import (
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type stubPasswordCredentialManager struct {
	identityAuthentication CanonicalPasswordAuthentication
	accountAuthentication  CanonicalPasswordAuthentication
	storedProvider         model.IdentityProvider
	storedSubject          string
	storedAccountSeq       int
	storeCalls             int
	replacedAccountSeq     int
	replaceCalls           int
}

func (s *stubPasswordCredentialManager) AuthenticateIdentity(model.IdentityProvider, string, string) (CanonicalPasswordAuthentication, error) {
	return s.identityAuthentication, nil
}

func (s *stubPasswordCredentialManager) AuthenticateAccount(int, string) (CanonicalPasswordAuthentication, error) {
	return s.accountAuthentication, nil
}

func (s *stubPasswordCredentialManager) StoreIdentityPassword(provider model.IdentityProvider, subject string, accountSeq int, _ string) error {
	s.storedProvider = provider
	s.storedSubject = subject
	s.storedAccountSeq = accountSeq
	s.storeCalls++
	return nil
}

func (s *stubPasswordCredentialManager) ReplaceAccountPassword(accountSeq int, _ string) error {
	s.replacedAccountSeq = accountSeq
	s.replaceCalls++
	return nil
}

type stubMemberRepository struct {
	user               *model.User
	legacyLoginCalls   int
	insertedLegacyHash string
}

func (s *stubMemberRepository) FindMemberByIDAndPwdAny(_ string, _ string) (*model.User, error) {
	s.legacyLoginCalls++
	return s.user, nil
}

func (s *stubMemberRepository) FindMemberByEmailAndPwdAny(_ string, _ string) (*model.User, error) {
	s.legacyLoginCalls++
	return s.user, nil
}

func (s *stubMemberRepository) GetMemberBySeq(int) (*model.User, error)       { return s.user, nil }
func (s *stubMemberRepository) FindMemberByPhone(string) (*model.User, error) { return s.user, nil }
func (s *stubMemberRepository) InsertMember(string, string, string, string, string, string, *int, string, string, string, string, string, string, string) (int, error) {
	return 42, nil
}

func TestMemberPasswordLoginUsesCanonicalCredentialFirst(t *testing.T) {
	memberRepo := &stubMemberRepository{user: &model.User{USRSeq: 42, USRStatus: "CCC"}}
	credentials := &stubPasswordCredentialManager{
		identityAuthentication: CanonicalPasswordAuthentication{State: CanonicalPasswordAuthenticated, AccountSeq: 42},
	}
	service := NewMemberServiceWithPasswordCredentials(memberRepo, credentials)

	user, err := service.LoginWithPassword("member-id", "submitted password")
	if err != nil {
		t.Fatalf("LoginWithPassword() error = %v", err)
	}
	if user == nil || user.USRSeq != 42 {
		t.Fatalf("LoginWithPassword() user = %#v", user)
	}
	if memberRepo.legacyLoginCalls != 0 {
		t.Fatalf("legacy login calls = %d, want 0", memberRepo.legacyLoginCalls)
	}
}

func TestMemberPasswordLoginBackfillsCanonicalCredentialAfterLegacyFallback(t *testing.T) {
	memberRepo := &stubMemberRepository{user: &model.User{USRSeq: 42, USRStatus: "CCC"}}
	credentials := &stubPasswordCredentialManager{}
	service := NewMemberServiceWithPasswordCredentials(memberRepo, credentials)

	user, err := service.LoginWithPassword("member-id", "submitted password")
	if err != nil {
		t.Fatalf("LoginWithPassword() error = %v", err)
	}
	if user == nil || memberRepo.legacyLoginCalls != 1 {
		t.Fatalf("user = %#v, legacy login calls = %d", user, memberRepo.legacyLoginCalls)
	}
	if credentials.storeCalls != 1 || credentials.storedProvider != model.IdentityProviderLocalUsername || credentials.storedSubject != "member-id" || credentials.storedAccountSeq != 42 {
		t.Fatalf("canonical backfill = %+v", credentials)
	}
}

func TestMemberPasswordLoginDoesNotFallbackWhenCanonicalRejectsPassword(t *testing.T) {
	memberRepo := &stubMemberRepository{user: &model.User{USRSeq: 42, USRStatus: "CCC"}}
	credentials := &stubPasswordCredentialManager{
		identityAuthentication: CanonicalPasswordAuthentication{State: CanonicalPasswordRejected, AccountSeq: 42},
	}
	service := NewMemberServiceWithPasswordCredentials(memberRepo, credentials)

	user, err := service.LoginWithPassword("member-id", "wrong password")
	if err != nil {
		t.Fatalf("LoginWithPassword() error = %v", err)
	}
	if user != nil || memberRepo.legacyLoginCalls != 0 || credentials.storeCalls != 0 {
		t.Fatalf("user = %#v, legacy calls = %d, canonical stores = %d", user, memberRepo.legacyLoginCalls, credentials.storeCalls)
	}
}

func TestMemberEmailPasswordLoginBackfillsCanonicalCredentialAfterLegacyFallback(t *testing.T) {
	memberRepo := &stubMemberRepository{user: &model.User{USRSeq: 42, USRStatus: "CCC"}}
	credentials := &stubPasswordCredentialManager{}
	service := NewMemberServiceWithPasswordCredentials(memberRepo, credentials)

	user, err := service.LoginWithEmailPassword("Member@Example.com", "submitted password")
	if err != nil {
		t.Fatalf("LoginWithEmailPassword() error = %v", err)
	}
	if user == nil || memberRepo.legacyLoginCalls != 1 {
		t.Fatalf("user = %#v, legacy calls = %d", user, memberRepo.legacyLoginCalls)
	}
	if credentials.storeCalls != 1 || credentials.storedProvider != model.IdentityProviderEmail || credentials.storedAccountSeq != 42 {
		t.Fatalf("canonical backfill = %+v", credentials)
	}
}

type stubRegistrationMemberRepository struct {
	stubMemberRepository
}

func (s *stubRegistrationMemberRepository) CheckIDExists(string) (bool, error)    { return false, nil }
func (s *stubRegistrationMemberRepository) CheckPhoneExists(string) (bool, error) { return false, nil }
func (s *stubRegistrationMemberRepository) CheckEmailExists(string) (bool, error) { return false, nil }
func (s *stubRegistrationMemberRepository) InsertMemberWithPwd(_ model.RegisterRequest, hash string) (int, error) {
	s.insertedLegacyHash = hash
	return 42, nil
}

type stubRegistrationProfileRepository struct{}

func (stubRegistrationProfileRepository) SaveUserTags(int, []string) error { return nil }

type stubPasswordSignupRepository struct {
	called     bool
	credential model.PasswordCredential
}

func (s *stubPasswordSignupRepository) CreatePasswordAccount(_ model.RegisterRequest, _ string, credential model.PasswordCredential) (int, error) {
	s.called = true
	s.credential = credential
	return 42, nil
}

func TestRegistrationStoresCanonicalAndLegacyPassword(t *testing.T) {
	memberRepo := &stubRegistrationMemberRepository{stubMemberRepository: stubMemberRepository{user: &model.User{USRSeq: 42}}}
	credentials := &stubPasswordCredentialManager{}
	service := NewRegistrationServiceWithPasswordCredentials(memberRepo, stubRegistrationProfileRepository{}, credentials)

	_, err := service.Register(model.RegisterRequest{UsrID: "new-member", Phone: "01012345678", Password: "Valid123!"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if memberRepo.insertedLegacyHash != MysqlNativePassword("Valid123!") {
		t.Fatal("Register() did not preserve the legacy password hash")
	}
	if credentials.storeCalls != 1 || credentials.storedProvider != model.IdentityProviderLocalUsername || credentials.storedSubject != "new-member" || credentials.storedAccountSeq != 42 {
		t.Fatalf("canonical password store = %+v", credentials)
	}
}

func TestTransactionalRegistrationDelegatesAllPasswordWritesToSignupRepository(t *testing.T) {
	memberRepo := &stubRegistrationMemberRepository{stubMemberRepository: stubMemberRepository{user: &model.User{USRSeq: 42}}}
	signupRepo := &stubPasswordSignupRepository{}
	service := NewTransactionalRegistrationService(memberRepo, stubRegistrationProfileRepository{}, signupRepo)

	_, err := service.Register(model.RegisterRequest{UsrID: "new-member", Phone: "01012345678", Password: "Valid123!", Tags: []string{"alumni"}})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if !signupRepo.called || signupRepo.credential.Algorithm != model.PasswordAlgorithmArgon2id {
		t.Fatalf("signup transaction = %+v", signupRepo)
	}
	if memberRepo.insertedLegacyHash != "" {
		t.Fatal("legacy member repository was called outside the signup transaction")
	}
}

type stubPasswordChangeRepository struct {
	storedHash  string
	updatedHash string
}

type stubAtomicPasswordChanger struct {
	called            bool
	legacyCurrent     string
	legacyReplacement string
	replacement       model.PasswordCredential
}

func (s *stubAtomicPasswordChanger) ChangePasswordAtomically(_ int, legacyCurrent, legacyReplacement string, replacement model.PasswordCredential, verify repository.PasswordCredentialVerifier) error {
	s.called = true
	s.legacyCurrent = legacyCurrent
	s.legacyReplacement = legacyReplacement
	s.replacement = replacement
	_, err := verify(model.PasswordCredential{
		Algorithm: model.PasswordAlgorithmMysqlNativePassword, PasswordHash: MysqlNativePassword("Current123!"), Status: model.PasswordCredentialStatusActive,
	})
	return err
}

func (s *stubPasswordChangeRepository) GetPasswordHash(int) (string, error) { return s.storedHash, nil }
func (s *stubPasswordChangeRepository) UpdatePassword(_ int, hash string) error {
	s.updatedHash = hash
	return nil
}

func TestPasswordChangeFallsBackToLegacyAndWritesBothStores(t *testing.T) {
	repo := &stubPasswordChangeRepository{storedHash: MysqlNativePassword("Current123!")}
	credentials := &stubPasswordCredentialManager{}
	service := NewPasswordChangeServiceWithPasswordCredentials(repo, credentials)

	err := service.ChangePassword(42, "Current123!", "Replacement123!")
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if credentials.replaceCalls != 1 || credentials.replacedAccountSeq != 42 {
		t.Fatalf("canonical replacement = %+v", credentials)
	}
	if repo.updatedHash != MysqlNativePassword("Replacement123!") {
		t.Fatal("ChangePassword() did not preserve the legacy fallback hash")
	}
}

func TestAtomicPasswordChangeDelegatesBothWritesToOneTransaction(t *testing.T) {
	repo := &stubAtomicPasswordChanger{}
	service := NewAtomicPasswordChangeService(repo)

	err := service.ChangePassword(42, "Current123!", "Replacement123!")
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if !repo.called || repo.legacyCurrent != MysqlNativePassword("Current123!") || repo.legacyReplacement != MysqlNativePassword("Replacement123!") {
		t.Fatalf("atomic legacy mutation = %+v", repo)
	}
	if repo.replacement.Algorithm != model.PasswordAlgorithmArgon2id || repo.replacement.PasswordHash == "" {
		t.Fatalf("atomic canonical replacement = %+v", repo.replacement)
	}
}
