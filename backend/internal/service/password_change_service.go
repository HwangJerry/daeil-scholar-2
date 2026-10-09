// password_change_service.go — Business logic for authenticated password changes (id/pw users).
package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// PasswordChangeService handles password mutation for logged-in id/pw users.
// It is distinct from PasswordResetService, which handles the forgotten-password flow.
type PasswordChangeService struct {
	repo        passwordChangeRepository
	credentials passwordCredentialManager
	atomicRepo  atomicPasswordChanger
	hasher      *PasswordHasher
}

type atomicPasswordChanger interface {
	ChangePasswordAtomically(int, string, string, model.PasswordCredential, repository.PasswordCredentialVerifier) error
}

type passwordChangeRepository interface {
	GetPasswordHash(int) (string, error)
	UpdatePassword(int, string) error
}

func NewPasswordChangeService(repo *repository.ProfileRepository) *PasswordChangeService {
	return &PasswordChangeService{repo: repo}
}

func NewPasswordChangeServiceWithPasswordCredentials(repo passwordChangeRepository, credentials passwordCredentialManager) *PasswordChangeService {
	return &PasswordChangeService{repo: repo, credentials: credentials}
}

func NewAtomicPasswordChangeService(repo atomicPasswordChanger) *PasswordChangeService {
	return &PasswordChangeService{atomicRepo: repo, hasher: NewPasswordHasher()}
}

// ChangePassword verifies currentPwd against the stored hash and replaces it with newPwd.
// Returns errors.New("NO_PASSWORD") for Kakao-only users and "WRONG_PASSWORD" on mismatch.
func (s *PasswordChangeService) ChangePassword(usrSeq int, currentPwd, newPwd string) error {
	if err := ValidateNewPassword(newPwd); err != nil {
		return err
	}
	if s.atomicRepo != nil {
		replacement, err := s.hasher.NewCredential(0, model.IdentityProviderLocalUsername, newPwd)
		if err != nil {
			return err
		}
		err = s.atomicRepo.ChangePasswordAtomically(
			usrSeq,
			MysqlNativePassword(currentPwd),
			MysqlNativePassword(newPwd),
			replacement,
			func(credential model.PasswordCredential) (bool, error) {
				verification, err := s.hasher.VerifyCredential(credential.Algorithm, currentPwd, credential.PasswordHash)
				return verification.Valid, err
			},
		)
		switch {
		case errors.Is(err, repository.ErrPasswordMissing):
			return errors.New("NO_PASSWORD")
		case errors.Is(err, repository.ErrPasswordMismatch), errors.Is(err, repository.ErrPasswordCredentialDisabled):
			return errors.New("WRONG_PASSWORD")
		default:
			return err
		}
	}

	authenticated := false
	if s.credentials != nil {
		authentication, err := s.credentials.AuthenticateAccount(usrSeq, currentPwd)
		if err != nil {
			return err
		}
		if authentication.State != CanonicalPasswordAbsent {
			if authentication.State != CanonicalPasswordAuthenticated {
				return errors.New("WRONG_PASSWORD")
			}
			authenticated = true
		}
	}
	if !authenticated {
		stored, err := s.repo.GetPasswordHash(usrSeq)
		if err != nil {
			return err
		}
		if stored == "" {
			return errors.New("NO_PASSWORD")
		}
		if stored != MysqlNativePassword(currentPwd) {
			return errors.New("WRONG_PASSWORD")
		}
	}
	if s.credentials != nil {
		if err := s.credentials.ReplaceAccountPassword(usrSeq, newPwd); err != nil {
			return err
		}
	}
	return s.repo.UpdatePassword(usrSeq, MysqlNativePassword(newPwd))
}
