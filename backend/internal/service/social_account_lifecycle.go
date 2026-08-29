package service

import (
	"context"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
)

type SocialAccountLifecycleService struct {
	auth     *AuthService
	apple    *AppleIdentityVerifier
	vault    *SocialCredentialVault
	vaultErr error
}

type AccountDeletionResult struct {
	RevocationPending bool
}

func NewSocialAccountLifecycleService(auth *AuthService, apple *AppleIdentityVerifier) *SocialAccountLifecycleService {
	vault, err := NewSocialCredentialVault(auth.cfg.Apple.CredentialEncryptionKey)
	return &SocialAccountLifecycleService{
		auth:     auth,
		apple:    apple,
		vault:    vault,
		vaultErr: err,
	}
}

func (s *SocialAccountLifecycleService) EnsureCredentialStorageAvailable(credential string) error {
	if credential == "" {
		return nil
	}
	if s.vaultErr != nil {
		return s.vaultErr
	}
	if !s.vault.Ready() {
		return ErrCredentialVaultNotConfigured
	}
	return nil
}

func (s *SocialAccountLifecycleService) StoreCredential(usrSeq int, provider model.SocialProvider, credential string) error {
	if credential == "" {
		return nil
	}
	if err := s.EnsureCredentialStorageAvailable(credential); err != nil {
		return err
	}
	encrypted, err := s.vault.Encrypt(credential)
	if err != nil {
		return err
	}
	return s.auth.repo.UpsertSocialCredential(usrSeq, string(provider), encrypted)
}

func (s *SocialAccountLifecycleService) EncryptCredential(credential string) (string, error) {
	if credential == "" {
		return "", nil
	}
	if err := s.EnsureCredentialStorageAvailable(credential); err != nil {
		return "", err
	}
	return s.vault.Encrypt(credential)
}

func (s *SocialAccountLifecycleService) DeleteAccount(_ context.Context, usrSeq int) (AccountDeletionResult, error) {
	err := s.auth.repo.AnonymizeAccountForDeletion(usrSeq)
	return AccountDeletionResult{}, err
}

func (s *SocialAccountLifecycleService) ApplyAppleNotification(notification AppleServerNotification) error {
	user, err := s.auth.FindMemberBySocialID(string(model.SocialProviderApple), notification.Subject)
	if err != nil || user == nil {
		return err
	}
	switch notification.Type {
	case "consent-revoked":
		if err := s.auth.repo.RevokeAllMobileSessions(user.USRSeq); err != nil {
			return err
		}
		if err := s.auth.repo.DeleteLegacySessionsByUser(user.USRSeq); err != nil {
			return err
		}
		return s.auth.repo.DeleteSocialConnection(user.USRSeq, string(model.SocialProviderApple))
	case "account-deleted":
		_, err := s.DeleteAccount(context.Background(), user.USRSeq)
		return err
	case "email-disabled":
		return s.auth.repo.UpdateSocialProviderEmailEnabled(user.USRSeq, string(model.SocialProviderApple), false)
	case "email-enabled":
		return s.auth.repo.UpdateSocialProviderEmailEnabled(user.USRSeq, string(model.SocialProviderApple), true)
	default:
		return errors.New("unsupported apple notification event")
	}
}
