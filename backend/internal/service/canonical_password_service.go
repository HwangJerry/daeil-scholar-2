package service

import (
	"errors"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
)

const argon2idAlgorithm = model.PasswordAlgorithmArgon2id

var ErrCanonicalPasswordIdentityMissing = errors.New("canonical password identity missing")

type passwordIdentityRepository interface {
	UpsertIdentity(model.Identity) error
	FindIdentityByProviderSubject(model.IdentityProvider, string) (*model.Identity, error)
	ListIdentities(int) ([]model.Identity, error)
}

type passwordCredentialRepository interface {
	UpsertPasswordCredential(model.PasswordCredential) error
	FindPasswordCredential(int64) (*model.PasswordCredential, error)
	RehashPasswordCredential(int64, string, model.PasswordCredential) (bool, error)
}

// CanonicalPasswordAuthentication distinguishes an absent canonical credential
// from a present credential whose submitted password is invalid.
type CanonicalPasswordAuthenticationState string

const (
	CanonicalPasswordAbsent        CanonicalPasswordAuthenticationState = ""
	CanonicalPasswordAuthenticated CanonicalPasswordAuthenticationState = "AUTHENTICATED"
	CanonicalPasswordRejected      CanonicalPasswordAuthenticationState = "REJECTED"
	CanonicalPasswordDisabled      CanonicalPasswordAuthenticationState = "DISABLED"
)

type CanonicalPasswordAuthentication struct {
	State      CanonicalPasswordAuthenticationState
	AccountSeq int
}

type passwordCredentialManager interface {
	AuthenticateIdentity(model.IdentityProvider, string, string) (CanonicalPasswordAuthentication, error)
	AuthenticateAccount(int, string) (CanonicalPasswordAuthentication, error)
	StoreIdentityPassword(model.IdentityProvider, string, int, string) error
	ReplaceAccountPassword(int, string) error
}

// CanonicalPasswordService owns canonical password credential transitions.
type CanonicalPasswordService struct {
	identities  passwordIdentityRepository
	credentials passwordCredentialRepository
	hasher      *PasswordHasher
}

func NewCanonicalPasswordService(identities passwordIdentityRepository, credentials passwordCredentialRepository) *CanonicalPasswordService {
	return &CanonicalPasswordService{
		identities:  identities,
		credentials: credentials,
		hasher:      NewPasswordHasher(),
	}
}

func (s *CanonicalPasswordService) AuthenticateIdentity(provider model.IdentityProvider, subject, password string) (CanonicalPasswordAuthentication, error) {
	subject = normalizePasswordIdentitySubject(provider, subject)
	identity, err := s.identities.FindIdentityByProviderSubject(provider, subject)
	if err != nil {
		return CanonicalPasswordAuthentication{}, err
	}
	if identity == nil {
		return CanonicalPasswordAuthentication{State: CanonicalPasswordAbsent}, nil
	}
	if identity.Status != model.IdentityStatusActive {
		return CanonicalPasswordAuthentication{State: CanonicalPasswordDisabled, AccountSeq: identity.AccountSeq}, nil
	}

	authentication, err := s.authenticateIdentity(*identity, password)
	if err != nil {
		return CanonicalPasswordAuthentication{}, err
	}
	return authentication, nil
}

func (s *CanonicalPasswordService) AuthenticateAccount(accountSeq int, password string) (CanonicalPasswordAuthentication, error) {
	identities, err := s.identities.ListIdentities(accountSeq)
	if err != nil {
		return CanonicalPasswordAuthentication{}, err
	}

	state := CanonicalPasswordAbsent
	for _, identity := range identities {
		if !identity.Provider.SupportsPassword() {
			continue
		}
		if identity.Status != model.IdentityStatusActive {
			state = CanonicalPasswordDisabled
			continue
		}
		authentication, err := s.authenticateIdentity(identity, password)
		if err != nil {
			return CanonicalPasswordAuthentication{}, err
		}
		if authentication.State == CanonicalPasswordAuthenticated {
			return authentication, nil
		}
		if authentication.State != CanonicalPasswordAbsent {
			state = authentication.State
		}
	}
	return CanonicalPasswordAuthentication{State: state, AccountSeq: accountSeq}, nil
}

func (s *CanonicalPasswordService) StoreIdentityPassword(provider model.IdentityProvider, subject string, accountSeq int, password string) error {
	if !provider.SupportsPassword() {
		return errors.New("password identity provider does not support passwords")
	}
	subject = normalizePasswordIdentitySubject(provider, subject)
	identity, err := s.identities.FindIdentityByProviderSubject(provider, subject)
	if err != nil {
		return err
	}
	if identity == nil {
		identityValue, err := newPasswordIdentity(accountSeq, provider, subject)
		if err != nil {
			return err
		}
		if err := s.identities.UpsertIdentity(identityValue); err != nil {
			return err
		}
		identity, err = s.identities.FindIdentityByProviderSubject(provider, subject)
		if err != nil {
			return err
		}
		if identity == nil {
			return errors.New("canonical password identity was not persisted")
		}
	}
	if identity.AccountSeq != accountSeq {
		return errors.New("canonical password identity belongs to another account")
	}
	if identity.Status != model.IdentityStatusActive {
		return errors.New("canonical password identity is not active")
	}
	return s.storeCredential(*identity, password)
}

func (s *CanonicalPasswordService) ReplaceAccountPassword(accountSeq int, password string) error {
	identities, err := s.identities.ListIdentities(accountSeq)
	if err != nil {
		return err
	}
	passwordIdentityCount := 0
	for _, identity := range identities {
		if !identity.Provider.SupportsPassword() || identity.Status != model.IdentityStatusActive {
			continue
		}
		passwordIdentityCount++
		if err := s.storeCredential(identity, password); err != nil {
			return err
		}
	}
	if passwordIdentityCount == 0 {
		return ErrCanonicalPasswordIdentityMissing
	}
	return nil
}

func (s *CanonicalPasswordService) authenticateIdentity(identity model.Identity, password string) (CanonicalPasswordAuthentication, error) {
	credential, err := s.credentials.FindPasswordCredential(identity.IdentityID)
	if err != nil {
		return CanonicalPasswordAuthentication{}, err
	}
	if credential == nil {
		return CanonicalPasswordAuthentication{State: CanonicalPasswordAbsent, AccountSeq: identity.AccountSeq}, nil
	}
	if credential.Status != model.PasswordCredentialStatusActive {
		return CanonicalPasswordAuthentication{State: CanonicalPasswordDisabled, AccountSeq: identity.AccountSeq}, nil
	}
	if credential.Provider != identity.Provider || !credential.Provider.SupportsPassword() {
		return CanonicalPasswordAuthentication{}, errors.New("canonical password credential provider mismatch")
	}
	verification, err := s.hasher.VerifyCredential(credential.Algorithm, password, credential.PasswordHash)
	if err != nil {
		return CanonicalPasswordAuthentication{}, err
	}
	authentication := CanonicalPasswordAuthentication{
		State:      CanonicalPasswordRejected,
		AccountSeq: identity.AccountSeq,
	}
	if verification.Valid {
		authentication.State = CanonicalPasswordAuthenticated
	}
	if verification.Valid && verification.NeedsRehash {
		credentialUpdate, err := s.newCredential(identity, password)
		if err != nil {
			return CanonicalPasswordAuthentication{}, err
		}
		updated, err := s.credentials.RehashPasswordCredential(identity.IdentityID, credential.PasswordHash, credentialUpdate)
		if err != nil {
			return CanonicalPasswordAuthentication{}, err
		}
		if !updated {
			return s.authenticateCurrentCredentialAfterRehashConflict(identity, password)
		}
	}
	return authentication, nil
}

func (s *CanonicalPasswordService) storeCredential(identity model.Identity, password string) error {
	credential, err := s.newCredential(identity, password)
	if err != nil {
		return err
	}
	return s.credentials.UpsertPasswordCredential(credential)
}

func (s *CanonicalPasswordService) newCredential(identity model.Identity, password string) (model.PasswordCredential, error) {
	return s.hasher.NewCredential(identity.IdentityID, identity.Provider, password)
}

func (s *CanonicalPasswordService) authenticateCurrentCredentialAfterRehashConflict(identity model.Identity, password string) (CanonicalPasswordAuthentication, error) {
	credential, err := s.credentials.FindPasswordCredential(identity.IdentityID)
	if err != nil {
		return CanonicalPasswordAuthentication{}, err
	}
	if credential == nil {
		return CanonicalPasswordAuthentication{State: CanonicalPasswordAbsent, AccountSeq: identity.AccountSeq}, nil
	}
	if credential.Status != model.PasswordCredentialStatusActive {
		return CanonicalPasswordAuthentication{State: CanonicalPasswordDisabled, AccountSeq: identity.AccountSeq}, nil
	}
	verification, err := s.hasher.VerifyCredential(credential.Algorithm, password, credential.PasswordHash)
	if err != nil {
		return CanonicalPasswordAuthentication{}, err
	}
	state := CanonicalPasswordRejected
	if verification.Valid {
		state = CanonicalPasswordAuthenticated
	}
	return CanonicalPasswordAuthentication{State: state, AccountSeq: identity.AccountSeq}, nil
}

func normalizePasswordIdentitySubject(provider model.IdentityProvider, subject string) string {
	subject = strings.TrimSpace(subject)
	if provider == model.IdentityProviderEmail {
		return strings.ToLower(subject)
	}
	return subject
}

func newPasswordIdentity(accountSeq int, provider model.IdentityProvider, subject string) (model.Identity, error) {
	if provider == model.IdentityProviderEmail {
		return model.NewIdentity(accountSeq, provider, subject, subject, true)
	}
	return model.NewIdentity(accountSeq, provider, subject, "", false)
}
