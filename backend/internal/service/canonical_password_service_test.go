package service

import (
	"bytes"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

type memoryPasswordIdentityRepository struct {
	identity   *model.Identity
	identities []model.Identity
}

func (r *memoryPasswordIdentityRepository) UpsertIdentity(identity model.Identity) error {
	identity.IdentityID = 101
	r.identity = &identity
	r.identities = append(r.identities, identity)
	return nil
}

func (r *memoryPasswordIdentityRepository) FindIdentityByProviderSubject(provider model.IdentityProvider, subject string) (*model.Identity, error) {
	if r.identity == nil || r.identity.Provider != provider || r.identity.SubjectKey != subject {
		return nil, nil
	}
	return r.identity, nil
}

func (r *memoryPasswordIdentityRepository) ListIdentities(int) ([]model.Identity, error) {
	return r.identities, nil
}

type memoryPasswordCredentialRepository struct {
	credential *model.PasswordCredential
	upserts    int
	rehashes   int
	casResult  bool
	casCurrent *model.PasswordCredential
}

func (r *memoryPasswordCredentialRepository) UpsertPasswordCredential(credential model.PasswordCredential) error {
	r.credential = &credential
	r.upserts++
	return nil
}

func (r *memoryPasswordCredentialRepository) FindPasswordCredential(identityID int64) (*model.PasswordCredential, error) {
	if r.credential == nil || r.credential.IdentityID != identityID {
		return nil, nil
	}
	return r.credential, nil
}

func (r *memoryPasswordCredentialRepository) RehashPasswordCredential(_ int64, _ string, credential model.PasswordCredential) (bool, error) {
	r.rehashes++
	if r.casResult {
		r.credential = &credential
		return true, nil
	}
	if r.casCurrent != nil {
		r.credential = r.casCurrent
	}
	return false, nil
}

func TestCanonicalPasswordServiceAuthenticatesCanonicalCredential(t *testing.T) {
	identity := model.Identity{IdentityID: 101, AccountSeq: 42, Provider: model.IdentityProviderLocalUsername, SubjectKey: "member", Status: model.IdentityStatusActive}
	hash, err := newPasswordHasher(defaultArgon2idParameters, bytes.NewReader(make([]byte, argon2idSaltLength))).Hash("canonical password")
	if err != nil {
		t.Fatal(err)
	}
	identities := &memoryPasswordIdentityRepository{identity: &identity, identities: []model.Identity{identity}}
	credentials := &memoryPasswordCredentialRepository{credential: &model.PasswordCredential{
		IdentityID: 101, Provider: model.IdentityProviderLocalUsername, Algorithm: argon2idAlgorithm, PasswordHash: hash, Status: model.PasswordCredentialStatusActive,
	}}
	service := NewCanonicalPasswordService(identities, credentials)

	authentication, err := service.AuthenticateIdentity(model.IdentityProviderLocalUsername, "member", "canonical password")
	if err != nil {
		t.Fatalf("AuthenticateIdentity() error = %v", err)
	}
	if authentication.State != CanonicalPasswordAuthenticated || authentication.AccountSeq != 42 {
		t.Fatalf("AuthenticateIdentity() = %+v", authentication)
	}
	if credentials.upserts != 0 {
		t.Fatalf("credential upserts = %d, want 0", credentials.upserts)
	}
}

func TestCanonicalPasswordServiceReportsMissingCredentialForLegacyFallback(t *testing.T) {
	identity := model.Identity{IdentityID: 101, AccountSeq: 42, Provider: model.IdentityProviderLocalUsername, SubjectKey: "member", Status: model.IdentityStatusActive}
	service := NewCanonicalPasswordService(
		&memoryPasswordIdentityRepository{identity: &identity, identities: []model.Identity{identity}},
		&memoryPasswordCredentialRepository{},
	)

	authentication, err := service.AuthenticateIdentity(model.IdentityProviderLocalUsername, "member", "submitted password")
	if err != nil {
		t.Fatalf("AuthenticateIdentity() error = %v", err)
	}
	if authentication.State != CanonicalPasswordAbsent || authentication.AccountSeq != 42 {
		t.Fatalf("AuthenticateIdentity() = %+v, want canonical-missing result", authentication)
	}
}

func TestCanonicalPasswordServiceRehashesOutdatedCredential(t *testing.T) {
	oldParameters := defaultArgon2idParameters
	oldParameters.Iterations--
	oldHash, err := newPasswordHasher(oldParameters, bytes.NewReader(make([]byte, argon2idSaltLength))).Hash("rehash password")
	if err != nil {
		t.Fatal(err)
	}
	identity := model.Identity{IdentityID: 101, AccountSeq: 42, Provider: model.IdentityProviderLocalUsername, SubjectKey: "member", Status: model.IdentityStatusActive}
	credentials := &memoryPasswordCredentialRepository{credential: &model.PasswordCredential{
		IdentityID: 101, Provider: model.IdentityProviderLocalUsername, Algorithm: argon2idAlgorithm, PasswordHash: oldHash, Status: model.PasswordCredentialStatusActive,
	}, casResult: true}
	service := NewCanonicalPasswordService(
		&memoryPasswordIdentityRepository{identity: &identity, identities: []model.Identity{identity}},
		credentials,
	)

	authentication, err := service.AuthenticateIdentity(model.IdentityProviderLocalUsername, "member", "rehash password")
	if err != nil {
		t.Fatalf("AuthenticateIdentity() error = %v", err)
	}
	if authentication.State != CanonicalPasswordAuthenticated || credentials.rehashes != 1 || credentials.upserts != 0 {
		t.Fatalf("authentication = %+v, credential rehashes = %d, upserts = %d", authentication, credentials.rehashes, credentials.upserts)
	}
	verification, err := NewPasswordHasher().Verify("rehash password", credentials.credential.PasswordHash)
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Valid || verification.NeedsRehash {
		t.Fatalf("persisted verification = %+v", verification)
	}
}

func TestCanonicalPasswordServiceRehashesAlgorithmTaggedLegacyCredential(t *testing.T) {
	identity := model.Identity{IdentityID: 101, AccountSeq: 42, Provider: model.IdentityProviderLocalUsername, SubjectKey: "member", Status: model.IdentityStatusActive}
	credentials := &memoryPasswordCredentialRepository{credential: &model.PasswordCredential{
		IdentityID: 101, Provider: identity.Provider, Algorithm: model.PasswordAlgorithmMysqlNativePassword,
		PasswordHash: MysqlNativePassword("legacy password"), Status: model.PasswordCredentialStatusActive,
	}, casResult: true}
	service := NewCanonicalPasswordService(&memoryPasswordIdentityRepository{identity: &identity}, credentials)

	authentication, err := service.AuthenticateIdentity(identity.Provider, identity.SubjectKey, "legacy password")
	if err != nil {
		t.Fatal(err)
	}
	if authentication.State != CanonicalPasswordAuthenticated || credentials.rehashes != 1 {
		t.Fatalf("authentication = %+v, rehashes = %d", authentication, credentials.rehashes)
	}
	if credentials.credential.Algorithm != model.PasswordAlgorithmArgon2id {
		t.Fatalf("algorithm = %q, want Argon2id", credentials.credential.Algorithm)
	}
}

func TestCanonicalPasswordServiceCreatesMissingIdentityAndCredential(t *testing.T) {
	identities := &memoryPasswordIdentityRepository{}
	credentials := &memoryPasswordCredentialRepository{}
	service := NewCanonicalPasswordService(identities, credentials)

	err := service.StoreIdentityPassword(model.IdentityProviderLocalUsername, "new-member", 42, "new password")
	if err != nil {
		t.Fatalf("StoreIdentityPassword() error = %v", err)
	}
	if identities.identity == nil || identities.identity.AccountSeq != 42 || credentials.credential == nil || credentials.credential.IdentityID != 101 {
		t.Fatalf("identity = %+v, credential = %+v", identities.identity, credentials.credential)
	}
}

func TestCanonicalPasswordServiceDoesNotFallbackForDisabledIdentity(t *testing.T) {
	identity := model.Identity{IdentityID: 101, AccountSeq: 42, Provider: model.IdentityProviderLocalUsername, SubjectKey: "member", Status: model.IdentityStatusDisabled}
	service := NewCanonicalPasswordService(
		&memoryPasswordIdentityRepository{identity: &identity, identities: []model.Identity{identity}},
		&memoryPasswordCredentialRepository{},
	)

	authentication, err := service.AuthenticateIdentity(model.IdentityProviderLocalUsername, "member", "submitted password")
	if err != nil {
		t.Fatal(err)
	}
	if authentication.State != CanonicalPasswordDisabled {
		t.Fatalf("authentication = %+v, want disabled", authentication)
	}
}

func TestCanonicalPasswordServiceDoesNotFallbackForDisabledCredential(t *testing.T) {
	identity := model.Identity{IdentityID: 101, AccountSeq: 42, Provider: model.IdentityProviderLocalUsername, SubjectKey: "member", Status: model.IdentityStatusActive}
	service := NewCanonicalPasswordService(
		&memoryPasswordIdentityRepository{identity: &identity, identities: []model.Identity{identity}},
		&memoryPasswordCredentialRepository{credential: &model.PasswordCredential{
			IdentityID: 101, Provider: model.IdentityProviderLocalUsername, Algorithm: model.PasswordAlgorithmArgon2id, PasswordHash: "disabled", Status: model.PasswordCredentialStatusDisabled,
		}},
	)

	authentication, err := service.AuthenticateIdentity(model.IdentityProviderLocalUsername, "member", "submitted password")
	if err != nil {
		t.Fatal(err)
	}
	if authentication.State != CanonicalPasswordDisabled {
		t.Fatalf("authentication = %+v, want disabled", authentication)
	}
}

func TestCanonicalPasswordServiceConcurrentRehashDoesNotRestoreOldPassword(t *testing.T) {
	oldParameters := defaultArgon2idParameters
	oldParameters.Iterations--
	oldHash, err := newPasswordHasher(oldParameters, bytes.NewReader(make([]byte, argon2idSaltLength))).Hash("old password")
	if err != nil {
		t.Fatal(err)
	}
	newHash, err := newPasswordHasher(defaultArgon2idParameters, bytes.NewReader(bytes.Repeat([]byte{1}, argon2idSaltLength))).Hash("new password")
	if err != nil {
		t.Fatal(err)
	}
	identity := model.Identity{IdentityID: 101, AccountSeq: 42, Provider: model.IdentityProviderLocalUsername, SubjectKey: "member", Status: model.IdentityStatusActive}
	credentials := &memoryPasswordCredentialRepository{
		credential: &model.PasswordCredential{IdentityID: 101, Provider: identity.Provider, Algorithm: model.PasswordAlgorithmArgon2id, PasswordHash: oldHash, Status: model.PasswordCredentialStatusActive},
		casCurrent: &model.PasswordCredential{IdentityID: 101, Provider: identity.Provider, Algorithm: model.PasswordAlgorithmArgon2id, PasswordHash: newHash, Status: model.PasswordCredentialStatusActive},
	}
	service := NewCanonicalPasswordService(&memoryPasswordIdentityRepository{identity: &identity}, credentials)

	authentication, err := service.AuthenticateIdentity(model.IdentityProviderLocalUsername, "member", "old password")
	if err != nil {
		t.Fatal(err)
	}
	if authentication.State != CanonicalPasswordRejected {
		t.Fatalf("authentication = %+v, want rejected after concurrent password change", authentication)
	}
	if credentials.credential.PasswordHash != newHash {
		t.Fatal("concurrent rehash replaced the new password")
	}
}
